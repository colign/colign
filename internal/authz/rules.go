package authz

// AuthRule maps an RPC method to a resource type and action for Casbin enforcement.
type AuthRule struct {
	Resource string
	Action   string
}

// rpcRules maps Connect RPC procedure names to their authorization rules.
// Any RPC not in this map or skipRPCs will be denied by default.
var rpcRules = map[string]AuthRule{
	// ProjectService
	"/project.v1.ProjectService/UpdateProject":       {Resource: "project", Action: "update"},
	"/project.v1.ProjectService/DeleteProject":       {Resource: "project", Action: "delete"},
	"/project.v1.ProjectService/InviteMember":        {Resource: "project", Action: "invite"},
	"/project.v1.ProjectService/AssignLabel":         {Resource: "project", Action: "assign_label"},
	"/project.v1.ProjectService/RemoveLabel":         {Resource: "project", Action: "remove_label"},
	"/project.v1.ProjectService/CreateChange":        {Resource: "change", Action: "create"},
	"/project.v1.ProjectService/ListChanges":         {Resource: "change", Action: "read"},
	"/project.v1.ProjectService/GetChange":           {Resource: "change", Action: "read"},
	"/project.v1.ProjectService/UpdateChange":        {Resource: "change", Action: "update"},
	"/project.v1.ProjectService/DeleteChange":        {Resource: "change", Action: "delete"},
	"/project.v1.ProjectService/ArchiveChange":       {Resource: "change", Action: "archive"},
	"/project.v1.ProjectService/UnarchiveChange":     {Resource: "change", Action: "unarchive"},
	"/project.v1.ProjectService/AssignChangeLabel":   {Resource: "change", Action: "update"},
	"/project.v1.ProjectService/RemoveChangeLabel":   {Resource: "change", Action: "update"},
	"/project.v1.ProjectService/GetArchivePolicy":    {Resource: "archive_policy", Action: "read"},
	"/project.v1.ProjectService/UpdateArchivePolicy": {Resource: "archive_policy", Action: "update"},

	// TaskService
	"/task.v1.TaskService/ListTasks":    {Resource: "task", Action: "read"},
	"/task.v1.TaskService/CreateTask":   {Resource: "task", Action: "create"},
	"/task.v1.TaskService/UpdateTask":   {Resource: "task", Action: "update"},
	"/task.v1.TaskService/DeleteTask":   {Resource: "task", Action: "delete"},
	"/task.v1.TaskService/ReorderTasks": {Resource: "task", Action: "reorder"},

	// WorkflowService
	"/workflow.v1.WorkflowService/GetStatus":         {Resource: "workflow", Action: "read"},
	"/workflow.v1.WorkflowService/Advance":           {Resource: "workflow", Action: "advance"},
	"/workflow.v1.WorkflowService/Approve":           {Resource: "workflow", Action: "approve"},
	"/workflow.v1.WorkflowService/RequestChanges":    {Resource: "workflow", Action: "revert"},
	"/workflow.v1.WorkflowService/Revert":            {Resource: "workflow", Action: "revert"},
	"/workflow.v1.WorkflowService/GetHistory":        {Resource: "workflow", Action: "read"},
	"/workflow.v1.WorkflowService/SetApprovalPolicy": {Resource: "workflow", Action: "set_policy"},

	// CommentService
	"/comment.v1.CommentService/CreateComment":  {Resource: "comment", Action: "create"},
	"/comment.v1.CommentService/ListComments":   {Resource: "comment", Action: "read"},
	"/comment.v1.CommentService/ResolveComment": {Resource: "comment", Action: "resolve"},
	"/comment.v1.CommentService/DeleteComment":  {Resource: "comment", Action: "delete"},
	"/comment.v1.CommentService/CreateReply":    {Resource: "comment", Action: "reply"},

	// DocumentService
	"/document.v1.DocumentService/GetDocument":  {Resource: "document", Action: "read"},
	"/document.v1.DocumentService/SaveDocument": {Resource: "document", Action: "save"},

	// AcceptanceCriteriaService
	"/acceptance.v1.AcceptanceCriteriaService/CreateAC": {Resource: "ac", Action: "create"},
	"/acceptance.v1.AcceptanceCriteriaService/ListAC":   {Resource: "ac", Action: "read"},
	"/acceptance.v1.AcceptanceCriteriaService/UpdateAC": {Resource: "ac", Action: "update"},
	"/acceptance.v1.AcceptanceCriteriaService/ToggleAC": {Resource: "ac", Action: "toggle"},
	"/acceptance.v1.AcceptanceCriteriaService/DeleteAC": {Resource: "ac", Action: "delete"},

	// MemoryService
	"/memory.v1.MemoryService/GetMemory":  {Resource: "memory", Action: "read"},
	"/memory.v1.MemoryService/SaveMemory": {Resource: "memory", Action: "save"},
}

// skipRPCs lists RPC procedures that bypass RBAC enforcement entirely.
// These are either non-project-scoped or handle their own authorization.
var skipRPCs = map[string]bool{
	// AuthService - pre-authentication
	"/auth.v1.AuthService/Register":          true,
	"/auth.v1.AuthService/Login":             true,
	"/auth.v1.AuthService/RefreshToken":      true,
	"/auth.v1.AuthService/GetUser":           true,
	"/auth.v1.AuthService/UpdateUser":        true,
	"/auth.v1.AuthService/ChangePassword":    true,
	"/auth.v1.AuthService/GetProviderStatus": true,

	// OrganizationService - org-scoped, not project-scoped
	"/organization.v1.OrganizationService/CreateOrganization": true,
	"/organization.v1.OrganizationService/GetOrganization":    true,
	"/organization.v1.OrganizationService/ListOrganizations":  true,
	"/organization.v1.OrganizationService/SwitchOrganization": true,
	"/organization.v1.OrganizationService/InviteMember":       true,
	"/organization.v1.OrganizationService/ListMembers":        true,
	"/organization.v1.OrganizationService/RemoveMember":       true,
	"/organization.v1.OrganizationService/UpdateMemberRole":   true,
	"/organization.v1.OrganizationService/ListInvitations":    true,
	"/organization.v1.OrganizationService/RevokeInvitation":   true,
	"/organization.v1.OrganizationService/UpdateOrganization": true,
	"/organization.v1.OrganizationService/DeleteOrganization": true,
	"/organization.v1.OrganizationService/AcceptInvitation":   true,

	// ApiTokenService - token management
	"/apitoken.v1.ApiTokenService/CreateToken": true,
	"/apitoken.v1.ApiTokenService/ListTokens":  true,
	"/apitoken.v1.ApiTokenService/DeleteToken": true,

	// ProjectService - non-project-scoped operations
	"/project.v1.ProjectService/CreateProject": true,
	"/project.v1.ProjectService/GetProject":    true,
	"/project.v1.ProjectService/ListProjects":  true,
	"/project.v1.ProjectService/CreateLabel":   true,
	"/project.v1.ProjectService/ListLabels":    true,
	"/project.v1.ProjectService/DeleteLabel":   true,
	"/project.v1.ProjectService/Search":        true,

	// NotificationService
	"/notification.v1.NotificationService/ListNotifications": true,
	"/notification.v1.NotificationService/MarkRead":          true,
	"/notification.v1.NotificationService/MarkAllRead":       true,
	"/notification.v1.NotificationService/GetUnreadCount":    true,
	"/notification.v1.NotificationService/Subscribe":         true,
}

// GetRule returns the auth rule for the given RPC procedure.
// Returns the rule and true if found, zero value and false if not.
func GetRule(procedure string) (AuthRule, bool) {
	rule, ok := rpcRules[procedure]
	return rule, ok
}

// IsSkipped returns true if the RPC procedure should bypass RBAC enforcement.
func IsSkipped(procedure string) bool {
	return skipRPCs[procedure]
}
