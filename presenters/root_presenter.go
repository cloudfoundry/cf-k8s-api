package presenters

type RootV3Presenter struct {
	Links RootV3PresenterLinks
}

type RootV3PresenterLinks struct {
	Self RootV3PresenterSelf
}

type RootV3PresenterSelf struct {
	Href string
}
