package menu

import "context"

type Repository interface {
	WithTx(context.Context, func(Repository) error) error
	Lock(context.Context) error
	List(context.Context) ([]Menu, error)
	Save(context.Context, *Menu) error
	Delete(context.Context, uint) error
}
