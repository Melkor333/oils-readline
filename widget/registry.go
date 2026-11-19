package widget

import (
	"github.com/Melkor333/oils-readline/shell"
)

type GlobalConstructor func(s shell.Shell) Model
type CommandConstructor func(cmd shell.Command) Model

type Registry struct {
	globalConstructors  []GlobalConstructor
	commandConstructors []CommandConstructor
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) RegisterGlobal(f GlobalConstructor) {
	r.globalConstructors = append(r.globalConstructors, f)
}

func (r *Registry) RegisterCommand(f CommandConstructor) {
	r.commandConstructors = append(r.commandConstructors, f)
}

func (r *Registry) CreateGlobals(s shell.Shell) []Model {
	models := make([]Model, len(r.globalConstructors))
	for i, f := range r.globalConstructors {
		models[i] = f(s)
	}
	return models
}

func (r *Registry) CreateForCommand(cmd shell.Command) []Model {
	models := make([]Model, len(r.commandConstructors))
	for i, f := range r.commandConstructors {
		models[i] = f(cmd)
	}
	return models
}
