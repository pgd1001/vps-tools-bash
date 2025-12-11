package app

type App struct {
	Name string
}

func New() *App {
	return &App{
		Name: "vps-tools",
	}
}