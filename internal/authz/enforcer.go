package authz

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"
)

const modelConf = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

const policyCSV = `
p, viewer, project, read
p, viewer, change, read
p, viewer, task, read
p, viewer, comment, read
p, viewer, document, read
p, viewer, memory, read
p, viewer, workflow, read
p, viewer, ac, read
p, viewer, archive_policy, read

p, editor, change, create
p, editor, change, update
p, editor, change, archive
p, editor, change, unarchive
p, editor, change, delete
p, editor, task, create
p, editor, task, update
p, editor, task, delete
p, editor, task, reorder
p, editor, comment, create
p, editor, comment, resolve
p, editor, comment, delete
p, editor, comment, reply
p, editor, document, save
p, editor, ac, create
p, editor, ac, update
p, editor, ac, toggle
p, editor, ac, delete
p, editor, memory, save
p, editor, workflow, advance
p, editor, workflow, revert
p, editor, workflow, approve

p, owner, project, update
p, owner, project, delete
p, owner, project, invite
p, owner, project, assign_label
p, owner, project, remove_label
p, owner, workflow, set_policy
p, owner, archive_policy, update

g, owner, editor
g, editor, viewer
`

// NewEnforcer creates a Casbin enforcer with the RBAC model and policy.
func NewEnforcer() (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, err
	}

	a := stringadapter.NewAdapter(policyCSV)

	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, err
	}

	return e, nil
}
