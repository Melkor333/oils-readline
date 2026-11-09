package widget

import (
	"github.com/Melkor333/oils-readline/shell"
)

type GlobalConstructor func(s shell.Shell) Model
type CommandConstructor func(cmd shell.Command) Model

type Registry struct {
	globals []GlobalConstructor
	perCmd  []CommandConstructor
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) RegisterGlobal(f GlobalConstructor) {
	r.globals = append(r.globals, f)
}

func (r *Registry) RegisterCommand(f CommandConstructor) {
	r.perCmd = append(r.perCmd, f)
}

func (r *Registry) CreateGlobals(s shell.Shell) []Model {
	models := make([]Model, len(r.globals))
	for i, f := range r.globals {
		models[i] = f(s)
	}
	return models
}

func (r *Registry) CreateForCommand(cmd shell.Command) []Model {
	models := make([]Model, len(r.perCmd))
	for i, f := range r.perCmd {
		models[i] = f(cmd)
	}
	return models
}
