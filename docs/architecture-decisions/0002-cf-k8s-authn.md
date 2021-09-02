# Authentication in cf on k8s will use the authentication method of the k8s cluster

## Status

Draft | *Under Review* | Accepted | Rejected | Deprecated | Superseded | Etc.

## Context

The [Cloud Foundry on Kubernetes: Authentication document](https://docs.google.com/document/u/0/d/1va6sE5uRi_iwGx5nzBcar145_jHq81i9SpwvpUdmsBA/edit), written before the proposal standard, details the EMEA team’s exploration into how we might perform authentication in cf on k8s.
Please refer to that document for full details.
Given that it should be officially reviewed, but we are going ahead with a proposal on how to modify the CF CLI to accommodate the authentication findings, we’ve created this ADR.
Below is a summary of the decision.

### Summary

#### Authentication Strategy

The guiding constraint was that authorization in cf on k8s should be handled natively by k8s as much as possible.
This means using standard k8s RBAC, involving (Cluster)Roles and (Cluster)RoleBindings.
Such bindings require a subject, which can be a k8s service account, a user or a group.

Service accounts are associated with non-expiring access tokens, since they are intended for non-interactive use by system components.
This makes them unsuitable for real users, whose application access we must be able to revoke easily and quickly.

Groups are a concept alien to CF.
We ‘cf set-org-role USERNAME ORG ROLE’, for example, rather than granting a group role permissions.

That leaves us with k8s user as the only sensible abstraction for the cf on k8s user.
K8s can be configured to use client certificates or tokens as user credentials.
Depending on the configuration, these will be validated within k8s, or by calling out to an external identity provider or helper.

The CF CLI will need to perform any necessary authentication with whatever system k8s is configured to use for user authentication to obtain a certificate or token.
So initially, the team attempted to standardize authentication so cf on k8s clusters would use a single common user authentication method, and the CF CLI would only need to be coded to authenticate with that single backend.
Quickly the team found that there was no practical common user authentication mechanism that would even work on the subset of GKE, AKS, EKS and Kind cluster types.

It is possible to standardize authentication at the k8s level by using a tool such as pinniped, but this itself delegates to an arbitrary OIDC backend identity provider to perform the authentication flow.
OIDC, although a standard, has many variations on the actual authentication flow, meaning pinniped does not actually give us the CF CLI authentication standardization we were seeking.

Finally, we realised kubectl somehow manages to authenticate correctly with any k8s cluster, regardless of its authentication configuration.
Code exists in the client-go library to use the kube config file to perform the necessary authentication flow, and is pluggable to accommodate any non-standard flow.

We showed how, given a working kube config, we can use the same client-go code as invoked by kubectl and perform any required authentication flow from the CF CLI.

#### CLI to Shim Authentication Passing

The CF CLI will be responsible for obtaining the user credentials to be used against the k8s API.
These will be passed to the cf-shim in each request, and the shim must use them to communicate with the k8s API as the user.
Note we are not authenticating communication with the shim.
We are passing credentials to the shim which uses them to authenticate with k8s.
For this reason, when client certificates are used, we need to pass both the certificate and the private key to the shim.

## Decision

The CF CLI will be extended to use k8s authentication based on configuration from the kube config file when used against cf on k8s.
If a user can authenticate with k8s via kubectl, then they can also authenticate with k8s via the CF CLI.

Authentication credentials will be passed to the shim in the form of an Authorization Header.
Basic auth will use a standard basic auth header (though we expect this not to be deprecated on most k8s clusters).
Auth tokens will use the standard bearer token auth header.
Client certificates / private key pairs will use a json- and base64- encoded object storing the client certificate and key.
We have suggested using the k8s ExecCredential struct for this, but this is not set in stone.

## Consequences

### Pros

- Cluster administrators can setup user authentication on their k8s clusters used for cf on k8s however they like
    - If they have existing identity providers used for single sign on, they can use them for cf on k8s
    - They can use the native identity provider for a hosted system, such as gcloud IAM for GKE or Azure AD for AKS.
- Any authentication configuration that works with kubectl will work with cf for k8s
- Authentication providers that can be integrated with k8s all have simple methods for populating a kube config
- `cf login` ceases to be a required step.
  Instead, the client-go code will check if valid credentials are available, and if so use them, or if they need refreshing, will perform the necessary background refresh, or if interactive authentication is required, will prompt for that.
- We hand off authentication flow code to a heavily used and maintained library.

### Cons

- A CF CLI user will require a working kube config to authenticate with cf on k8s.
- Although using public elements from the client-go authentication modules, the Credentials Plugin code feels slightly unstable, and the integration might need to be modified as we keep the client-go library up to date.
  However the Credential Plugin has just GA’ed, so maybe it will stabilize now.

### Other considerations

When the CF CLI is using UAA as its backend, it has the ability to manage users.
You can add or remove users via the CF CLI.
With cf on k8s, we now have an interface with the identity provider that only allows authentication and credential fetching.
User management is not defined and must be performed directly on the backend identity provider.
This could be seen as a security benefit as it keeps organization level user management to a single place.

Security concerns may be raised about passing a private key from the CF CLI to the shim.
These are normal left on the client.
However, these private keys are regenerated for each new client certificate, and only exist to prove ownership of the certificate.
The certificate has a short lifetime, maybe only 5 minutes, in the case of pinniped for example.
Thus the key makes the certificate useful as an authentication mechanism, but only for the lifespan of the certificate.
In these terms, the certificate and key pair are an equivalent security risk to a token, and no more.
