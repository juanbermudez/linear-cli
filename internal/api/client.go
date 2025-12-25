package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/juanbermudez/agent-linear-cli/internal/auth"
)

const (
	// LinearAPIEndpoint is the Linear GraphQL API endpoint
	LinearAPIEndpoint = "https://api.linear.app/graphql"
)

// Client is the Linear API client
type Client struct {
	graphql    *graphql.Client
	httpClient *http.Client
}

// NewClient creates a new Linear API client using the auth manager
func NewClient(ctx context.Context) (*Client, error) {
	manager := auth.NewManager()
	token, _, err := manager.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	return NewClientWithToken(token), nil
}

// NewClientWithToken creates a new Linear API client with a specific token
func NewClientWithToken(token string) *Client {
	httpClient := &http.Client{
		Transport: &authTransport{
			token: token,
			base:  http.DefaultTransport,
		},
	}

	return &Client{
		graphql:    graphql.NewClient(LinearAPIEndpoint, httpClient),
		httpClient: httpClient,
	}
}

// authTransport adds the Authorization header to all requests
type authTransport struct {
	token string
	base  http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.token)
	req.Header.Set("Content-Type", "application/json")
	return t.base.RoundTrip(req)
}

// Query executes a GraphQL query
func (c *Client) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	return c.graphql.Query(ctx, q, variables)
}

// Mutate executes a GraphQL mutation
func (c *Client) Mutate(ctx context.Context, m interface{}, variables map[string]interface{}) error {
	return c.graphql.Mutate(ctx, m, variables)
}

// Viewer represents the authenticated user
type Viewer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Active      bool   `json:"active"`
	Admin       bool   `json:"admin"`
	AvatarUrl   string `json:"avatarUrl,omitempty"`
}

// Organization represents a Linear organization
type Organization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UrlKey    string `json:"urlKey"`
	LogoUrl   string `json:"logoUrl,omitempty"`
}

// ViewerResponse is the response for viewer query
type ViewerResponse struct {
	Viewer       Viewer       `json:"viewer"`
	Organization Organization `json:"organization"`
}

// GetViewer fetches the authenticated user's information
func (c *Client) GetViewer(ctx context.Context) (*ViewerResponse, error) {
	var query struct {
		Viewer struct {
			ID          string `graphql:"id"`
			Name        string `graphql:"name"`
			DisplayName string `graphql:"displayName"`
			Email       string `graphql:"email"`
			Active      bool   `graphql:"active"`
			Admin       bool   `graphql:"admin"`
			AvatarUrl   string `graphql:"avatarUrl"`
		} `graphql:"viewer"`
		Organization struct {
			ID      string `graphql:"id"`
			Name    string `graphql:"name"`
			UrlKey  string `graphql:"urlKey"`
			LogoUrl string `graphql:"logoUrl"`
		} `graphql:"organization"`
	}

	if err := c.Query(ctx, &query, nil); err != nil {
		return nil, err
	}

	return &ViewerResponse{
		Viewer: Viewer{
			ID:          query.Viewer.ID,
			Name:        query.Viewer.Name,
			DisplayName: query.Viewer.DisplayName,
			Email:       query.Viewer.Email,
			Active:      query.Viewer.Active,
			Admin:       query.Viewer.Admin,
			AvatarUrl:   query.Viewer.AvatarUrl,
		},
		Organization: Organization{
			ID:      query.Organization.ID,
			Name:    query.Organization.Name,
			UrlKey:  query.Organization.UrlKey,
			LogoUrl: query.Organization.LogoUrl,
		},
	}, nil
}

// Issue represents a Linear issue
type Issue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority"`
	State       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"state"`
	Assignee *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"assignee,omitempty"`
	Project *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project,omitempty"`
	Labels struct {
		Nodes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	URL       string `json:"url"`
}

// Project represents a Linear project
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	State       string `json:"state"`
	Progress    int    `json:"progress"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	URL         string `json:"url"`
}

// Document represents a Linear document
type Document struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
	SlugID    string `json:"slugId"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Creator   *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"creator,omitempty"`
	Project *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project,omitempty"`
}

// Team represents a Linear team
type Team struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// User represents a Linear user
type User struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Active      bool   `json:"active"`
	Admin       bool   `json:"admin"`
}

// WorkflowState represents a workflow state
type WorkflowState struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	Color    string `json:"color"`
}

// Label represents a Linear label
type Label struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	ParentID string `json:"parentId,omitempty"`
}

// IssueState represents an issue's workflow state
type IssueState struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
}

// IssueAssignee represents an issue assignee
type IssueAssignee struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// IssueLabel represents a label on an issue
type IssueLabel struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// IssueTeam represents the team an issue belongs to
type IssueTeam struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// IssueProject represents the project an issue belongs to
type IssueProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IssueParent represents a parent issue
type IssueParent struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
}

// IssueChild represents a child issue
type IssueChild struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		Name string `json:"name"`
	} `json:"state"`
}

// IssueRelation represents a relationship between issues
type IssueRelation struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	RelatedIssue struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
	} `json:"relatedIssue"`
}

// IssueCycle represents a cycle
type IssueCycle struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

// IssueMilestone represents a project milestone
type IssueMilestone struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TargetDate string `json:"targetDate,omitempty"`
}

// Comment represents an issue comment
type Comment struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	User      *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"user,omitempty"`
	Parent *struct {
		ID string `json:"id"`
	} `json:"parent,omitempty"`
}

// IssueDetail represents a full issue with all details
type IssueDetail struct {
	ID               string          `json:"id"`
	Identifier       string          `json:"identifier"`
	Title            string          `json:"title"`
	Description      string          `json:"description,omitempty"`
	URL              string          `json:"url"`
	BranchName       string          `json:"branchName,omitempty"`
	Priority         int             `json:"priority"`
	Estimate         *float64        `json:"estimate,omitempty"`
	DueDate          string          `json:"dueDate,omitempty"`
	CreatedAt        string          `json:"createdAt"`
	UpdatedAt        string          `json:"updatedAt"`
	State            IssueState      `json:"state"`
	Assignee         *IssueAssignee  `json:"assignee,omitempty"`
	Team             IssueTeam       `json:"team"`
	Project          *IssueProject   `json:"project,omitempty"`
	ProjectMilestone *IssueMilestone `json:"projectMilestone,omitempty"`
	Cycle            *IssueCycle     `json:"cycle,omitempty"`
	Parent           *IssueParent    `json:"parent,omitempty"`
	Children         []IssueChild    `json:"children,omitempty"`
	Relations        []IssueRelation `json:"relations,omitempty"`
	Labels           []IssueLabel    `json:"labels,omitempty"`
	Comments         []Comment       `json:"comments,omitempty"`
}

// IssueListItem represents an issue in a list
type IssueListItem struct {
	ID         string         `json:"id"`
	Identifier string         `json:"identifier"`
	Title      string         `json:"title"`
	Priority   int            `json:"priority"`
	Estimate   *float64       `json:"estimate,omitempty"`
	State      IssueState     `json:"state"`
	Assignee   *IssueAssignee `json:"assignee,omitempty"`
	Labels     []IssueLabel   `json:"labels,omitempty"`
	UpdatedAt  string         `json:"updatedAt"`
}

// IssuesResponse is the response for issues list
type IssuesResponse struct {
	Issues []IssueListItem `json:"issues"`
	Count  int             `json:"count"`
}

// IssueCreateInput represents input for creating an issue
type IssueCreateInput struct {
	Title              string   `json:"title"`
	TeamID             string   `json:"teamId"`
	Description        string   `json:"description,omitempty"`
	AssigneeID         string   `json:"assigneeId,omitempty"`
	Priority           *int     `json:"priority,omitempty"`
	Estimate           *float64 `json:"estimate,omitempty"`
	DueDate            string   `json:"dueDate,omitempty"`
	LabelIDs           []string `json:"labelIds,omitempty"`
	ProjectID          string   `json:"projectId,omitempty"`
	StateID            string   `json:"stateId,omitempty"`
	ParentID           string   `json:"parentId,omitempty"`
	CycleID            string   `json:"cycleId,omitempty"`
	ProjectMilestoneID string   `json:"projectMilestoneId,omitempty"`
}

// IssueUpdateInput represents input for updating an issue
type IssueUpdateInput struct {
	Title              string   `json:"title,omitempty"`
	Description        string   `json:"description,omitempty"`
	AssigneeID         string   `json:"assigneeId,omitempty"`
	Priority           *int     `json:"priority,omitempty"`
	Estimate           *float64 `json:"estimate,omitempty"`
	DueDate            string   `json:"dueDate,omitempty"`
	LabelIDs           []string `json:"labelIds,omitempty"`
	ProjectID          string   `json:"projectId,omitempty"`
	StateID            string   `json:"stateId,omitempty"`
	ParentID           string   `json:"parentId,omitempty"`
	CycleID            string   `json:"cycleId,omitempty"`
	ProjectMilestoneID string   `json:"projectMilestoneId,omitempty"`
}

// IssueCreateResponse is the response for creating an issue
type IssueCreateResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Identifier string `json:"identifier"`
	URL     string `json:"url"`
	TeamKey string `json:"teamKey"`
}

// SearchIssuesResponse is the response for issue search
type SearchIssuesResponse struct {
	Issues     []IssueListItem `json:"issues"`
	TotalCount int             `json:"totalCount"`
	HasMore    bool            `json:"hasMore"`
	Query      string          `json:"query"`
}

// TeamsResponse is the response for teams query
type TeamsResponse struct {
	Teams []Team `json:"teams"`
	Count int    `json:"count"`
}

// GetTeams fetches all teams in the workspace
func (c *Client) GetTeams(ctx context.Context) (*TeamsResponse, error) {
	var query struct {
		Teams struct {
			Nodes []struct {
				ID         string  `graphql:"id"`
				Key        string  `graphql:"key"`
				Name       string  `graphql:"name"`
				Color      string  `graphql:"color"`
				ArchivedAt *string `graphql:"archivedAt"`
			} `graphql:"nodes"`
		} `graphql:"teams"`
	}

	if err := c.Query(ctx, &query, nil); err != nil {
		return nil, err
	}

	// Filter out archived teams (archivedAt is nil for active teams)
	teams := make([]Team, 0)
	for _, t := range query.Teams.Nodes {
		if t.ArchivedAt == nil {
			teams = append(teams, Team{
				ID:   t.ID,
				Key:  t.Key,
				Name: t.Name,
			})
		}
	}

	return &TeamsResponse{
		Teams: teams,
		Count: len(teams),
	}, nil
}

// GetTeamByKey fetches a team by its key
func (c *Client) GetTeamByKey(ctx context.Context, key string) (*Team, error) {
	var query struct {
		Teams struct {
			Nodes []struct {
				ID   string `graphql:"id"`
				Key  string `graphql:"key"`
				Name string `graphql:"name"`
			} `graphql:"nodes"`
		} `graphql:"teams(filter: {key: {eq: $key}})"`
	}

	variables := map[string]interface{}{
		"key": key,
	}

	if err := c.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	if len(query.Teams.Nodes) == 0 {
		return nil, nil
	}

	t := query.Teams.Nodes[0]
	return &Team{
		ID:   t.ID,
		Key:  t.Key,
		Name: t.Name,
	}, nil
}

// UsersResponse is the response for users query
type UsersResponse struct {
	Users []User `json:"users"`
	Count int    `json:"count"`
}

// GetUsers fetches all users in the workspace
func (c *Client) GetUsers(ctx context.Context) (*UsersResponse, error) {
	var query struct {
		Users struct {
			Nodes []struct {
				ID          string `graphql:"id"`
				Name        string `graphql:"name"`
				DisplayName string `graphql:"displayName"`
				Email       string `graphql:"email"`
				Active      bool   `graphql:"active"`
				Admin       bool   `graphql:"admin"`
			} `graphql:"nodes"`
		} `graphql:"users"`
	}

	if err := c.Query(ctx, &query, nil); err != nil {
		return nil, err
	}

	users := make([]User, len(query.Users.Nodes))
	for i, u := range query.Users.Nodes {
		users[i] = User{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: u.DisplayName,
			Email:       u.Email,
			Active:      u.Active,
			Admin:       u.Admin,
		}
	}

	return &UsersResponse{
		Users: users,
		Count: len(users),
	}, nil
}

// WorkflowStatesResponse is the response for workflow states query
type WorkflowStatesResponse struct {
	WorkflowStates []WorkflowState `json:"workflowStates"`
	Count          int             `json:"count"`
}

// GetWorkflowStates fetches workflow states for a team
func (c *Client) GetWorkflowStates(ctx context.Context, teamID string) (*WorkflowStatesResponse, error) {
	var query struct {
		Team struct {
			States struct {
				Nodes []struct {
					ID       string  `graphql:"id"`
					Name     string  `graphql:"name"`
					Type     string  `graphql:"type"`
					Position float64 `graphql:"position"`
					Color    string  `graphql:"color"`
				} `graphql:"nodes"`
			} `graphql:"states"`
		} `graphql:"team(id: $teamId)"`
	}

	variables := map[string]interface{}{
		"teamId": teamID,
	}

	if err := c.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	states := make([]WorkflowState, len(query.Team.States.Nodes))
	for i, s := range query.Team.States.Nodes {
		states[i] = WorkflowState{
			ID:       s.ID,
			Name:     s.Name,
			Type:     s.Type,
			Position: int(s.Position),
			Color:    s.Color,
		}
	}

	return &WorkflowStatesResponse{
		WorkflowStates: states,
		Count:          len(states),
	}, nil
}

// LabelsResponse is the response for labels query
type LabelsResponse struct {
	Labels []Label `json:"labels"`
	Count  int     `json:"count"`
}

// GetLabels fetches labels for a team
func (c *Client) GetLabels(ctx context.Context, teamID string) (*LabelsResponse, error) {
	var query struct {
		Team struct {
			Labels struct {
				Nodes []struct {
					ID          string `graphql:"id"`
					Name        string `graphql:"name"`
					Color       string `graphql:"color"`
					Description string `graphql:"description"`
					Parent      *struct {
						ID string `graphql:"id"`
					} `graphql:"parent"`
				} `graphql:"nodes"`
			} `graphql:"labels"`
		} `graphql:"team(id: $teamId)"`
	}

	variables := map[string]interface{}{
		"teamId": teamID,
	}

	if err := c.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	labels := make([]Label, len(query.Team.Labels.Nodes))
	for i, l := range query.Team.Labels.Nodes {
		labels[i] = Label{
			ID:    l.ID,
			Name:  l.Name,
			Color: l.Color,
		}
		if l.Parent != nil {
			labels[i].ParentID = l.Parent.ID
		}
	}

	return &LabelsResponse{
		Labels: labels,
		Count:  len(labels),
	}, nil
}

// IssueFilter contains filters for listing issues
type IssueFilter struct {
	TeamID     string
	StateTypes []string // triage, backlog, unstarted, started, completed, canceled
	AssigneeID string
	Unassigned bool
	ProjectID  string
}

// GetIssues fetches issues with filters
func (c *Client) GetIssues(ctx context.Context, filter IssueFilter, limit int, sortBy string) (*IssuesResponse, error) {
	// Build filter conditions for the query
	filterParts := []string{}

	if filter.TeamID != "" {
		filterParts = append(filterParts, fmt.Sprintf(`team: { id: { eq: "%s" } }`, filter.TeamID))
	}

	if len(filter.StateTypes) > 0 {
		types := ""
		for i, t := range filter.StateTypes {
			if i > 0 {
				types += ", "
			}
			types += fmt.Sprintf(`"%s"`, t)
		}
		filterParts = append(filterParts, fmt.Sprintf(`state: { type: { in: [%s] } }`, types))
	}

	if filter.Unassigned {
		filterParts = append(filterParts, `assignee: { null: true }`)
	} else if filter.AssigneeID != "" {
		filterParts = append(filterParts, fmt.Sprintf(`assignee: { id: { eq: "%s" } }`, filter.AssigneeID))
	}

	if filter.ProjectID != "" {
		filterParts = append(filterParts, fmt.Sprintf(`project: { id: { eq: "%s" } }`, filter.ProjectID))
	}

	// Build the filter string
	filterStr := ""
	if len(filterParts) > 0 {
		filterStr = ", filter: { "
		for i, part := range filterParts {
			if i > 0 {
				filterStr += ", "
			}
			filterStr += part
		}
		filterStr += " }"
	}

	// Build the raw GraphQL query
	queryStr := fmt.Sprintf(`query {
		issues(first: %d%s) {
			nodes {
				id
				identifier
				title
				priority
				estimate
				updatedAt
				state {
					id
					name
					type
					color
				}
				assignee {
					id
					name
					displayName
				}
				labels {
					nodes {
						id
						name
						color
					}
				}
			}
		}
	}`, limit, filterStr)

	// Execute raw query
	var result struct {
		Issues struct {
			Nodes []struct {
				ID         string  `json:"id"`
				Identifier string  `json:"identifier"`
				Title      string  `json:"title"`
				Priority   int     `json:"priority"`
				Estimate   float64 `json:"estimate"`
				UpdatedAt  string  `json:"updatedAt"`
				State      struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Type  string `json:"type"`
					Color string `json:"color"`
				} `json:"state"`
				Assignee *struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
				Labels struct {
					Nodes []struct {
						ID    string `json:"id"`
						Name  string `json:"name"`
						Color string `json:"color"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"nodes"`
		} `json:"issues"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	issues := make([]IssueListItem, len(result.Issues.Nodes))
	for i, issue := range result.Issues.Nodes {
		issues[i] = IssueListItem{
			ID:         issue.ID,
			Identifier: issue.Identifier,
			Title:      issue.Title,
			Priority:   issue.Priority,
			UpdatedAt:  issue.UpdatedAt,
			State: IssueState{
				ID:    issue.State.ID,
				Name:  issue.State.Name,
				Type:  issue.State.Type,
				Color: issue.State.Color,
			},
		}
		if issue.Estimate > 0 {
			est := issue.Estimate
			issues[i].Estimate = &est
		}
		if issue.Assignee != nil {
			issues[i].Assignee = &IssueAssignee{
				ID:          issue.Assignee.ID,
				Name:        issue.Assignee.Name,
				DisplayName: issue.Assignee.DisplayName,
			}
		}
		labels := make([]IssueLabel, len(issue.Labels.Nodes))
		for j, label := range issue.Labels.Nodes {
			labels[j] = IssueLabel{
				ID:    label.ID,
				Name:  label.Name,
				Color: label.Color,
			}
		}
		issues[i].Labels = labels
	}

	return &IssuesResponse{
		Issues: issues,
		Count:  len(issues),
	}, nil
}

// GetIssue fetches a single issue by ID or identifier
func (c *Client) GetIssue(ctx context.Context, issueID string, includeComments bool) (*IssueDetail, error) {
	var query struct {
		Issue struct {
			ID          string  `graphql:"id"`
			Identifier  string  `graphql:"identifier"`
			Title       string  `graphql:"title"`
			Description string  `graphql:"description"`
			URL         string  `graphql:"url"`
			BranchName  string  `graphql:"branchName"`
			Priority    int     `graphql:"priority"`
			Estimate    float64 `graphql:"estimate"`
			DueDate     string  `graphql:"dueDate"`
			CreatedAt   string  `graphql:"createdAt"`
			UpdatedAt   string  `graphql:"updatedAt"`
			State       struct {
				ID    string `graphql:"id"`
				Name  string `graphql:"name"`
				Type  string `graphql:"type"`
				Color string `graphql:"color"`
			} `graphql:"state"`
			Assignee *struct {
				ID          string `graphql:"id"`
				Name        string `graphql:"name"`
				DisplayName string `graphql:"displayName"`
			} `graphql:"assignee"`
			Team struct {
				ID   string `graphql:"id"`
				Key  string `graphql:"key"`
				Name string `graphql:"name"`
			} `graphql:"team"`
			Project *struct {
				ID   string `graphql:"id"`
				Name string `graphql:"name"`
			} `graphql:"project"`
			ProjectMilestone *struct {
				ID         string `graphql:"id"`
				Name       string `graphql:"name"`
				TargetDate string `graphql:"targetDate"`
			} `graphql:"projectMilestone"`
			Cycle *struct {
				ID       string `graphql:"id"`
				Name     string `graphql:"name"`
				StartsAt string `graphql:"startsAt"`
				EndsAt   string `graphql:"endsAt"`
			} `graphql:"cycle"`
			Parent *struct {
				ID         string `graphql:"id"`
				Identifier string `graphql:"identifier"`
				Title      string `graphql:"title"`
			} `graphql:"parent"`
			Children struct {
				Nodes []struct {
					ID         string `graphql:"id"`
					Identifier string `graphql:"identifier"`
					Title      string `graphql:"title"`
					State      struct {
						Name string `graphql:"name"`
					} `graphql:"state"`
				} `graphql:"nodes"`
			} `graphql:"children"`
			Relations struct {
				Nodes []struct {
					ID   string `graphql:"id"`
					Type string `graphql:"type"`
					RelatedIssue struct {
						ID         string `graphql:"id"`
						Identifier string `graphql:"identifier"`
						Title      string `graphql:"title"`
					} `graphql:"relatedIssue"`
				} `graphql:"nodes"`
			} `graphql:"relations"`
			Labels struct {
				Nodes []struct {
					ID    string `graphql:"id"`
					Name  string `graphql:"name"`
					Color string `graphql:"color"`
				} `graphql:"nodes"`
			} `graphql:"labels"`
		} `graphql:"issue(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": issueID,
	}

	if err := c.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	issue := &IssueDetail{
		ID:          query.Issue.ID,
		Identifier:  query.Issue.Identifier,
		Title:       query.Issue.Title,
		Description: query.Issue.Description,
		URL:         query.Issue.URL,
		BranchName:  query.Issue.BranchName,
		Priority:    query.Issue.Priority,
		DueDate:     query.Issue.DueDate,
		CreatedAt:   query.Issue.CreatedAt,
		UpdatedAt:   query.Issue.UpdatedAt,
		State: IssueState{
			ID:    query.Issue.State.ID,
			Name:  query.Issue.State.Name,
			Type:  query.Issue.State.Type,
			Color: query.Issue.State.Color,
		},
		Team: IssueTeam{
			ID:   query.Issue.Team.ID,
			Key:  query.Issue.Team.Key,
			Name: query.Issue.Team.Name,
		},
	}

	if query.Issue.Estimate > 0 {
		est := query.Issue.Estimate
		issue.Estimate = &est
	}

	if query.Issue.Assignee != nil {
		issue.Assignee = &IssueAssignee{
			ID:          query.Issue.Assignee.ID,
			Name:        query.Issue.Assignee.Name,
			DisplayName: query.Issue.Assignee.DisplayName,
		}
	}

	if query.Issue.Project != nil {
		issue.Project = &IssueProject{
			ID:   query.Issue.Project.ID,
			Name: query.Issue.Project.Name,
		}
	}

	if query.Issue.ProjectMilestone != nil {
		issue.ProjectMilestone = &IssueMilestone{
			ID:         query.Issue.ProjectMilestone.ID,
			Name:       query.Issue.ProjectMilestone.Name,
			TargetDate: query.Issue.ProjectMilestone.TargetDate,
		}
	}

	if query.Issue.Cycle != nil {
		issue.Cycle = &IssueCycle{
			ID:       query.Issue.Cycle.ID,
			Name:     query.Issue.Cycle.Name,
			StartsAt: query.Issue.Cycle.StartsAt,
			EndsAt:   query.Issue.Cycle.EndsAt,
		}
	}

	if query.Issue.Parent != nil {
		issue.Parent = &IssueParent{
			ID:         query.Issue.Parent.ID,
			Identifier: query.Issue.Parent.Identifier,
			Title:      query.Issue.Parent.Title,
		}
	}

	for _, child := range query.Issue.Children.Nodes {
		issue.Children = append(issue.Children, IssueChild{
			ID:         child.ID,
			Identifier: child.Identifier,
			Title:      child.Title,
			State:      struct{ Name string `json:"name"` }{Name: child.State.Name},
		})
	}

	for _, rel := range query.Issue.Relations.Nodes {
		issue.Relations = append(issue.Relations, IssueRelation{
			ID:   rel.ID,
			Type: rel.Type,
			RelatedIssue: struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
			}{
				ID:         rel.RelatedIssue.ID,
				Identifier: rel.RelatedIssue.Identifier,
				Title:      rel.RelatedIssue.Title,
			},
		})
	}

	for _, label := range query.Issue.Labels.Nodes {
		issue.Labels = append(issue.Labels, IssueLabel{
			ID:    label.ID,
			Name:  label.Name,
			Color: label.Color,
		})
	}

	// Fetch comments separately if requested
	if includeComments {
		comments, err := c.GetIssueComments(ctx, issueID, 50)
		if err == nil {
			issue.Comments = comments
		}
	}

	return issue, nil
}

// GetIssueComments fetches comments for an issue
func (c *Client) GetIssueComments(ctx context.Context, issueID string, limit int) ([]Comment, error) {
	var query struct {
		Issue struct {
			Comments struct {
				Nodes []struct {
					ID        string `graphql:"id"`
					Body      string `graphql:"body"`
					CreatedAt string `graphql:"createdAt"`
					User      *struct {
						ID          string `graphql:"id"`
						Name        string `graphql:"name"`
						DisplayName string `graphql:"displayName"`
					} `graphql:"user"`
					Parent *struct {
						ID string `graphql:"id"`
					} `graphql:"parent"`
				} `graphql:"nodes"`
			} `graphql:"comments(first: $limit)"`
		} `graphql:"issue(id: $id)"`
	}

	variables := map[string]interface{}{
		"id":    issueID,
		"limit": limit,
	}

	if err := c.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	comments := make([]Comment, len(query.Issue.Comments.Nodes))
	for i, c := range query.Issue.Comments.Nodes {
		comments[i] = Comment{
			ID:        c.ID,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		}
		if c.User != nil {
			comments[i].User = &struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			}{
				ID:          c.User.ID,
				Name:        c.User.Name,
				DisplayName: c.User.DisplayName,
			}
		}
		if c.Parent != nil {
			comments[i].Parent = &struct {
				ID string `json:"id"`
			}{ID: c.Parent.ID}
		}
	}

	return comments, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(ctx context.Context, input IssueCreateInput) (*IssueCreateResponse, error) {
	// Build input fields for the mutation
	inputParts := []string{
		fmt.Sprintf(`title: %q`, input.Title),
		fmt.Sprintf(`teamId: %q`, input.TeamID),
	}

	if input.Description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, input.Description))
	}
	if input.AssigneeID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`assigneeId: %q`, input.AssigneeID))
	}
	if input.Priority != nil {
		inputParts = append(inputParts, fmt.Sprintf(`priority: %d`, *input.Priority))
	}
	if input.Estimate != nil {
		inputParts = append(inputParts, fmt.Sprintf(`estimate: %v`, *input.Estimate))
	}
	if input.DueDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`dueDate: %q`, input.DueDate))
	}
	if len(input.LabelIDs) > 0 {
		labels := ""
		for i, id := range input.LabelIDs {
			if i > 0 {
				labels += ", "
			}
			labels += fmt.Sprintf(`%q`, id)
		}
		inputParts = append(inputParts, fmt.Sprintf(`labelIds: [%s]`, labels))
	}
	if input.ProjectID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`projectId: %q`, input.ProjectID))
	}
	if input.StateID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`stateId: %q`, input.StateID))
	}
	if input.ParentID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`parentId: %q`, input.ParentID))
	}
	if input.CycleID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`cycleId: %q`, input.CycleID))
	}
	if input.ProjectMilestoneID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`projectMilestoneId: %q`, input.ProjectMilestoneID))
	}

	// Build input string
	inputStr := ""
	for i, part := range inputParts {
		if i > 0 {
			inputStr += ", "
		}
		inputStr += part
	}

	mutationStr := fmt.Sprintf(`mutation {
		issueCreate(input: { %s }) {
			success
			issue {
				id
				identifier
				url
				team {
					key
				}
			}
		}
	}`, inputStr)

	var result struct {
		IssueCreate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				URL        string `json:"url"`
				Team       struct {
					Key string `json:"key"`
				} `json:"team"`
			} `json:"issue"`
		} `json:"issueCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.IssueCreate.Success {
		return nil, fmt.Errorf("failed to create issue")
	}

	return &IssueCreateResponse{
		Success:    true,
		ID:         result.IssueCreate.Issue.ID,
		Identifier: result.IssueCreate.Issue.Identifier,
		URL:        result.IssueCreate.Issue.URL,
		TeamKey:    result.IssueCreate.Issue.Team.Key,
	}, nil
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(ctx context.Context, issueID string, input IssueUpdateInput) (*IssueCreateResponse, error) {
	// Build input fields for the mutation
	inputParts := []string{}

	if input.Title != "" {
		inputParts = append(inputParts, fmt.Sprintf(`title: %q`, input.Title))
	}
	if input.Description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, input.Description))
	}
	if input.AssigneeID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`assigneeId: %q`, input.AssigneeID))
	}
	if input.Priority != nil {
		inputParts = append(inputParts, fmt.Sprintf(`priority: %d`, *input.Priority))
	}
	if input.Estimate != nil {
		inputParts = append(inputParts, fmt.Sprintf(`estimate: %v`, *input.Estimate))
	}
	if input.DueDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`dueDate: %q`, input.DueDate))
	}
	if len(input.LabelIDs) > 0 {
		labels := ""
		for i, id := range input.LabelIDs {
			if i > 0 {
				labels += ", "
			}
			labels += fmt.Sprintf(`%q`, id)
		}
		inputParts = append(inputParts, fmt.Sprintf(`labelIds: [%s]`, labels))
	}
	if input.ProjectID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`projectId: %q`, input.ProjectID))
	}
	if input.StateID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`stateId: %q`, input.StateID))
	}
	if input.ParentID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`parentId: %q`, input.ParentID))
	}
	if input.CycleID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`cycleId: %q`, input.CycleID))
	}
	if input.ProjectMilestoneID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`projectMilestoneId: %q`, input.ProjectMilestoneID))
	}

	if len(inputParts) == 0 {
		return nil, fmt.Errorf("at least one field must be provided to update")
	}

	// Build input string
	inputStr := ""
	for i, part := range inputParts {
		if i > 0 {
			inputStr += ", "
		}
		inputStr += part
	}

	mutationStr := fmt.Sprintf(`mutation {
		issueUpdate(id: %q, input: { %s }) {
			success
			issue {
				id
				identifier
				url
				team {
					key
				}
			}
		}
	}`, issueID, inputStr)

	var result struct {
		IssueUpdate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				URL        string `json:"url"`
				Team       struct {
					Key string `json:"key"`
				} `json:"team"`
			} `json:"issue"`
		} `json:"issueUpdate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.IssueUpdate.Success {
		return nil, fmt.Errorf("failed to update issue")
	}

	return &IssueCreateResponse{
		Success:    true,
		ID:         result.IssueUpdate.Issue.ID,
		Identifier: result.IssueUpdate.Issue.Identifier,
		URL:        result.IssueUpdate.Issue.URL,
		TeamKey:    result.IssueUpdate.Issue.Team.Key,
	}, nil
}

// DeleteIssue deletes an issue
func (c *Client) DeleteIssue(ctx context.Context, issueID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		issueDelete(id: %q) {
			success
		}
	}`, issueID)

	var result struct {
		IssueDelete struct {
			Success bool `json:"success"`
		} `json:"issueDelete"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.IssueDelete.Success {
		return fmt.Errorf("failed to delete issue")
	}

	return nil
}

// SearchIssues searches for issues
func (c *Client) SearchIssues(ctx context.Context, term string, limit int, includeArchived bool, teamID string) (*SearchIssuesResponse, error) {
	queryStr := fmt.Sprintf(`query {
		searchIssues(term: %q, first: %d, includeArchived: %t) {
			nodes {
				id
				identifier
				title
				priority
				estimate
				createdAt
				updatedAt
				state {
					id
					name
					type
					color
				}
				assignee {
					id
					name
					displayName
				}
				team {
					key
					name
				}
			}
			pageInfo {
				hasNextPage
			}
			totalCount
		}
	}`, term, limit, includeArchived)

	var result struct {
		SearchIssues struct {
			Nodes []struct {
				ID         string  `json:"id"`
				Identifier string  `json:"identifier"`
				Title      string  `json:"title"`
				Priority   int     `json:"priority"`
				Estimate   float64 `json:"estimate"`
				CreatedAt  string  `json:"createdAt"`
				UpdatedAt  string  `json:"updatedAt"`
				State      struct {
					ID    string `json:"id"`
					Name  string `json:"name"`
					Type  string `json:"type"`
					Color string `json:"color"`
				} `json:"state"`
				Assignee *struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
				Team struct {
					Key  string `json:"key"`
					Name string `json:"name"`
				} `json:"team"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool `json:"hasNextPage"`
			} `json:"pageInfo"`
			TotalCount int `json:"totalCount"`
		} `json:"searchIssues"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	issues := make([]IssueListItem, len(result.SearchIssues.Nodes))
	for i, issue := range result.SearchIssues.Nodes {
		issues[i] = IssueListItem{
			ID:         issue.ID,
			Identifier: issue.Identifier,
			Title:      issue.Title,
			Priority:   issue.Priority,
			UpdatedAt:  issue.UpdatedAt,
			State: IssueState{
				ID:    issue.State.ID,
				Name:  issue.State.Name,
				Type:  issue.State.Type,
				Color: issue.State.Color,
			},
		}
		if issue.Estimate > 0 {
			est := issue.Estimate
			issues[i].Estimate = &est
		}
		if issue.Assignee != nil {
			issues[i].Assignee = &IssueAssignee{
				ID:          issue.Assignee.ID,
				Name:        issue.Assignee.Name,
				DisplayName: issue.Assignee.DisplayName,
			}
		}
	}

	return &SearchIssuesResponse{
		Issues:     issues,
		TotalCount: result.SearchIssues.TotalCount,
		HasMore:    result.SearchIssues.PageInfo.HasNextPage,
		Query:      term,
	}, nil
}

// CreateComment creates a comment on an issue
func (c *Client) CreateComment(ctx context.Context, issueID string, body string) (*Comment, error) {
	mutationStr := fmt.Sprintf(`mutation {
		commentCreate(input: { issueId: %q, body: %q }) {
			success
			comment {
				id
				body
				createdAt
				user {
					id
					name
					displayName
				}
			}
		}
	}`, issueID, body)

	var result struct {
		CommentCreate struct {
			Success bool `json:"success"`
			Comment struct {
				ID        string `json:"id"`
				Body      string `json:"body"`
				CreatedAt string `json:"createdAt"`
				User      *struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"comment"`
		} `json:"commentCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.CommentCreate.Success {
		return nil, fmt.Errorf("failed to create comment")
	}

	comment := &Comment{
		ID:        result.CommentCreate.Comment.ID,
		Body:      result.CommentCreate.Comment.Body,
		CreatedAt: result.CommentCreate.Comment.CreatedAt,
	}

	if result.CommentCreate.Comment.User != nil {
		comment.User = &struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		}{
			ID:          result.CommentCreate.Comment.User.ID,
			Name:        result.CommentCreate.Comment.User.Name,
			DisplayName: result.CommentCreate.Comment.User.DisplayName,
		}
	}

	return comment, nil
}

// CreateIssueRelation creates a relationship between issues
func (c *Client) CreateIssueRelation(ctx context.Context, issueID, relatedIssueID, relationType string) error {
	mutationStr := fmt.Sprintf(`mutation {
		issueRelationCreate(input: { issueId: %q, relatedIssueId: %q, type: %s }) {
			success
		}
	}`, issueID, relatedIssueID, relationType)

	var result struct {
		IssueRelationCreate struct {
			Success bool `json:"success"`
		} `json:"issueRelationCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.IssueRelationCreate.Success {
		return fmt.Errorf("failed to create issue relation")
	}

	return nil
}

// DeleteIssueRelation removes a relationship between issues
func (c *Client) DeleteIssueRelation(ctx context.Context, relationID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		issueRelationDelete(id: %q) {
			success
		}
	}`, relationID)

	var result struct {
		IssueRelationDelete struct {
			Success bool `json:"success"`
		} `json:"issueRelationDelete"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.IssueRelationDelete.Success {
		return fmt.Errorf("failed to delete issue relation")
	}

	return nil
}

// GetViewer returns the current authenticated user (needed for "self" assignee)
func (c *Client) GetViewerID(ctx context.Context) (string, error) {
	viewer, err := c.GetViewer(ctx)
	if err != nil {
		return "", err
	}
	return viewer.Viewer.ID, nil
}

// Attachment represents an issue attachment
type Attachment struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	Subtitle  *string `json:"subtitle,omitempty"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	Creator   *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"creator,omitempty"`
}

// AttachmentsResponse is the response for listing attachments
type AttachmentsResponse struct {
	Attachments []Attachment `json:"attachments"`
	Count       int          `json:"count"`
}

// GetIssueAttachments fetches attachments for an issue
func (c *Client) GetIssueAttachments(ctx context.Context, issueID string) (*AttachmentsResponse, error) {
	queryStr := fmt.Sprintf(`query {
		issue(id: %q) {
			attachments {
				nodes {
					id
					title
					url
					subtitle
					createdAt
					updatedAt
					creator {
						id
						name
						displayName
					}
				}
			}
		}
	}`, issueID)

	var result struct {
		Issue struct {
			Attachments struct {
				Nodes []Attachment `json:"nodes"`
			} `json:"attachments"`
		} `json:"issue"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	return &AttachmentsResponse{
		Attachments: result.Issue.Attachments.Nodes,
		Count:       len(result.Issue.Attachments.Nodes),
	}, nil
}

// CreateAttachment creates a new attachment on an issue
func (c *Client) CreateAttachment(ctx context.Context, issueID, title, url string, subtitle *string) (*Attachment, error) {
	subtitlePart := ""
	if subtitle != nil && *subtitle != "" {
		subtitlePart = fmt.Sprintf(`, subtitle: %q`, *subtitle)
	}

	mutationStr := fmt.Sprintf(`mutation {
		attachmentCreate(input: { issueId: %q, title: %q, url: %q%s }) {
			success
			attachment {
				id
				title
				url
				subtitle
				createdAt
				updatedAt
			}
		}
	}`, issueID, title, url, subtitlePart)

	var result struct {
		AttachmentCreate struct {
			Success    bool       `json:"success"`
			Attachment Attachment `json:"attachment"`
		} `json:"attachmentCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.AttachmentCreate.Success {
		return nil, fmt.Errorf("failed to create attachment")
	}

	return &result.AttachmentCreate.Attachment, nil
}

// DeleteAttachment deletes an attachment
func (c *Client) DeleteAttachment(ctx context.Context, attachmentID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		attachmentDelete(id: %q) {
			success
		}
	}`, attachmentID)

	var result struct {
		AttachmentDelete struct {
			Success bool `json:"success"`
		} `json:"attachmentDelete"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.AttachmentDelete.Success {
		return fmt.Errorf("failed to delete attachment")
	}

	return nil
}

// ProjectDetail represents a detailed project
type ProjectDetail struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Content     string  `json:"content,omitempty"`
	SlugID      string  `json:"slugId"`
	Icon        string  `json:"icon,omitempty"`
	Color       string  `json:"color,omitempty"`
	State       string  `json:"state"`
	Progress    float64 `json:"progress"`
	StartDate   string  `json:"startDate,omitempty"`
	TargetDate  string  `json:"targetDate,omitempty"`
	URL         string  `json:"url"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	Status      *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"status,omitempty"`
	Lead *struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	} `json:"lead,omitempty"`
	Teams []struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"teams,omitempty"`
}

// ProjectListItem represents a project in a list
type ProjectListItem struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	SlugID     string  `json:"slugId"`
	State      string  `json:"state"`
	Progress   float64 `json:"progress"`
	TargetDate string  `json:"targetDate,omitempty"`
	URL        string  `json:"url"`
	UpdatedAt  string  `json:"updatedAt"`
	Status     *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"status,omitempty"`
	Lead *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"lead,omitempty"`
	Teams []struct {
		Key string `json:"key"`
	} `json:"teams,omitempty"`
}

// ProjectsResponse is the response for listing projects
type ProjectsResponse struct {
	Projects []ProjectListItem `json:"projects"`
	Count    int               `json:"count"`
}

// ProjectCreateInput is the input for creating a project
type ProjectCreateInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Content     string   `json:"content,omitempty"`
	TeamIDs     []string `json:"teamIds"`
	StatusID    string   `json:"statusId,omitempty"`
	LeadID      string   `json:"leadId,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Color       string   `json:"color,omitempty"`
	StartDate   string   `json:"startDate,omitempty"`
	TargetDate  string   `json:"targetDate,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
}

// ProjectUpdateInput is the input for updating a project
type ProjectUpdateInput struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	StatusID    string `json:"statusId,omitempty"`
	LeadID      string `json:"leadId,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	TargetDate  string `json:"targetDate,omitempty"`
	Priority    *int   `json:"priority,omitempty"`
}

// GetProjects fetches projects
func (c *Client) GetProjects(ctx context.Context, teamID string, limit int) (*ProjectsResponse, error) {
	filterPart := ""
	if teamID != "" {
		filterPart = fmt.Sprintf(`, filter: { teams: { id: { eq: "%s" } } }`, teamID)
	}

	queryStr := fmt.Sprintf(`query {
		projects(first: %d%s) {
			nodes {
				id
				name
				slugId
				state
				progress
				targetDate
				url
				updatedAt
				status {
					id
					name
					type
				}
				lead {
					id
					displayName
				}
				teams {
					nodes {
						key
					}
				}
			}
		}
	}`, limit, filterPart)

	var result struct {
		Projects struct {
			Nodes []struct {
				ID         string  `json:"id"`
				Name       string  `json:"name"`
				SlugID     string  `json:"slugId"`
				State      string  `json:"state"`
				Progress   float64 `json:"progress"`
				TargetDate string  `json:"targetDate"`
				URL        string  `json:"url"`
				UpdatedAt  string  `json:"updatedAt"`
				Status     *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"status"`
				Lead *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"lead"`
				Teams struct {
					Nodes []struct {
						Key string `json:"key"`
					} `json:"nodes"`
				} `json:"teams"`
			} `json:"nodes"`
		} `json:"projects"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	projects := make([]ProjectListItem, len(result.Projects.Nodes))
	for i, p := range result.Projects.Nodes {
		teams := make([]struct {
			Key string `json:"key"`
		}, len(p.Teams.Nodes))
		for j, t := range p.Teams.Nodes {
			teams[j] = struct {
				Key string `json:"key"`
			}{Key: t.Key}
		}
		projects[i] = ProjectListItem{
			ID:         p.ID,
			Name:       p.Name,
			SlugID:     p.SlugID,
			State:      p.State,
			Progress:   p.Progress,
			TargetDate: p.TargetDate,
			URL:        p.URL,
			UpdatedAt:  p.UpdatedAt,
			Status:     p.Status,
			Lead:       p.Lead,
			Teams:      teams,
		}
	}

	return &ProjectsResponse{
		Projects: projects,
		Count:    len(projects),
	}, nil
}

// GetProject fetches a single project by ID
func (c *Client) GetProject(ctx context.Context, projectID string) (*ProjectDetail, error) {
	queryStr := fmt.Sprintf(`query {
		project(id: %q) {
			id
			name
			description
			content
			slugId
			icon
			color
			state
			progress
			startDate
			targetDate
			url
			createdAt
			updatedAt
			status {
				id
				name
				type
			}
			lead {
				id
				name
				displayName
			}
			teams {
				nodes {
					id
					key
					name
				}
			}
		}
	}`, projectID)

	var result struct {
		Project struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Content     string  `json:"content"`
			SlugID      string  `json:"slugId"`
			Icon        string  `json:"icon"`
			Color       string  `json:"color"`
			State       string  `json:"state"`
			Progress    float64 `json:"progress"`
			StartDate   string  `json:"startDate"`
			TargetDate  string  `json:"targetDate"`
			URL         string  `json:"url"`
			CreatedAt   string  `json:"createdAt"`
			UpdatedAt   string  `json:"updatedAt"`
			Status      *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"status"`
			Lead *struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			} `json:"lead"`
			Teams struct {
				Nodes []struct {
					ID   string `json:"id"`
					Key  string `json:"key"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"teams"`
		} `json:"project"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	project := &ProjectDetail{
		ID:          result.Project.ID,
		Name:        result.Project.Name,
		Description: result.Project.Description,
		Content:     result.Project.Content,
		SlugID:      result.Project.SlugID,
		Icon:        result.Project.Icon,
		Color:       result.Project.Color,
		State:       result.Project.State,
		Progress:    result.Project.Progress,
		StartDate:   result.Project.StartDate,
		TargetDate:  result.Project.TargetDate,
		URL:         result.Project.URL,
		CreatedAt:   result.Project.CreatedAt,
		UpdatedAt:   result.Project.UpdatedAt,
		Status:      result.Project.Status,
		Lead:        result.Project.Lead,
	}

	teams := make([]struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	}, len(result.Project.Teams.Nodes))
	for i, t := range result.Project.Teams.Nodes {
		teams[i] = struct {
			ID   string `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
		}{ID: t.ID, Key: t.Key, Name: t.Name}
	}
	project.Teams = teams

	return project, nil
}

// CreateProject creates a new project
func (c *Client) CreateProject(ctx context.Context, input ProjectCreateInput) (*ProjectDetail, error) {
	// Build input parts
	inputParts := []string{
		fmt.Sprintf(`name: %q`, input.Name),
	}

	if len(input.TeamIDs) > 0 {
		teamIDs := ""
		for i, id := range input.TeamIDs {
			if i > 0 {
				teamIDs += ", "
			}
			teamIDs += fmt.Sprintf(`%q`, id)
		}
		inputParts = append(inputParts, fmt.Sprintf(`teamIds: [%s]`, teamIDs))
	}

	if input.Description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, input.Description))
	}
	if input.Content != "" {
		inputParts = append(inputParts, fmt.Sprintf(`content: %q`, input.Content))
	}
	if input.StatusID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`statusId: %q`, input.StatusID))
	}
	if input.LeadID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`leadId: %q`, input.LeadID))
	}
	if input.Icon != "" {
		inputParts = append(inputParts, fmt.Sprintf(`icon: %q`, input.Icon))
	}
	if input.Color != "" {
		inputParts = append(inputParts, fmt.Sprintf(`color: %q`, input.Color))
	}
	if input.StartDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`startDate: %q`, input.StartDate))
	}
	if input.TargetDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`targetDate: %q`, input.TargetDate))
	}
	if input.Priority != nil {
		inputParts = append(inputParts, fmt.Sprintf(`priority: %d`, *input.Priority))
	}

	inputStr := ""
	for i, part := range inputParts {
		if i > 0 {
			inputStr += ", "
		}
		inputStr += part
	}

	mutationStr := fmt.Sprintf(`mutation {
		projectCreate(input: { %s }) {
			success
			project {
				id
				name
				slugId
				url
				state
				status {
					id
					name
					type
				}
				lead {
					id
					name
					displayName
				}
				teams {
					nodes {
						id
						key
						name
					}
				}
			}
		}
	}`, inputStr)

	var result struct {
		ProjectCreate struct {
			Success bool `json:"success"`
			Project struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				SlugID string `json:"slugId"`
				URL    string `json:"url"`
				State  string `json:"state"`
				Status *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"status"`
				Lead *struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"lead"`
				Teams struct {
					Nodes []struct {
						ID   string `json:"id"`
						Key  string `json:"key"`
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"teams"`
			} `json:"project"`
		} `json:"projectCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.ProjectCreate.Success {
		return nil, fmt.Errorf("failed to create project")
	}

	project := &ProjectDetail{
		ID:     result.ProjectCreate.Project.ID,
		Name:   result.ProjectCreate.Project.Name,
		SlugID: result.ProjectCreate.Project.SlugID,
		URL:    result.ProjectCreate.Project.URL,
		State:  result.ProjectCreate.Project.State,
		Status: result.ProjectCreate.Project.Status,
		Lead:   result.ProjectCreate.Project.Lead,
	}

	teams := make([]struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	}, len(result.ProjectCreate.Project.Teams.Nodes))
	for i, t := range result.ProjectCreate.Project.Teams.Nodes {
		teams[i] = struct {
			ID   string `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
		}{ID: t.ID, Key: t.Key, Name: t.Name}
	}
	project.Teams = teams

	return project, nil
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(ctx context.Context, projectID string, input ProjectUpdateInput) (*ProjectDetail, error) {
	inputParts := []string{}

	if input.Name != "" {
		inputParts = append(inputParts, fmt.Sprintf(`name: %q`, input.Name))
	}
	if input.Description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, input.Description))
	}
	if input.Content != "" {
		inputParts = append(inputParts, fmt.Sprintf(`content: %q`, input.Content))
	}
	if input.StatusID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`statusId: %q`, input.StatusID))
	}
	if input.LeadID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`leadId: %q`, input.LeadID))
	}
	if input.Icon != "" {
		inputParts = append(inputParts, fmt.Sprintf(`icon: %q`, input.Icon))
	}
	if input.Color != "" {
		inputParts = append(inputParts, fmt.Sprintf(`color: %q`, input.Color))
	}
	if input.StartDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`startDate: %q`, input.StartDate))
	}
	if input.TargetDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`targetDate: %q`, input.TargetDate))
	}
	if input.Priority != nil {
		inputParts = append(inputParts, fmt.Sprintf(`priority: %d`, *input.Priority))
	}

	if len(inputParts) == 0 {
		return nil, fmt.Errorf("at least one field must be provided to update")
	}

	inputStr := ""
	for i, part := range inputParts {
		if i > 0 {
			inputStr += ", "
		}
		inputStr += part
	}

	mutationStr := fmt.Sprintf(`mutation {
		projectUpdate(id: %q, input: { %s }) {
			success
			project {
				id
				name
				slugId
				url
				state
			}
		}
	}`, projectID, inputStr)

	var result struct {
		ProjectUpdate struct {
			Success bool `json:"success"`
			Project struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				SlugID string `json:"slugId"`
				URL    string `json:"url"`
				State  string `json:"state"`
			} `json:"project"`
		} `json:"projectUpdate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.ProjectUpdate.Success {
		return nil, fmt.Errorf("failed to update project")
	}

	return &ProjectDetail{
		ID:     result.ProjectUpdate.Project.ID,
		Name:   result.ProjectUpdate.Project.Name,
		SlugID: result.ProjectUpdate.Project.SlugID,
		URL:    result.ProjectUpdate.Project.URL,
		State:  result.ProjectUpdate.Project.State,
	}, nil
}

// DeleteProject archives a project
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		projectArchive(id: %q) {
			success
		}
	}`, projectID)

	var result struct {
		ProjectArchive struct {
			Success bool `json:"success"`
		} `json:"projectArchive"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.ProjectArchive.Success {
		return fmt.Errorf("failed to archive project")
	}

	return nil
}

// RestoreProject unarchives a project
func (c *Client) RestoreProject(ctx context.Context, projectID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		projectUnarchive(id: %q) {
			success
		}
	}`, projectID)

	var result struct {
		ProjectUnarchive struct {
			Success bool `json:"success"`
		} `json:"projectUnarchive"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.ProjectUnarchive.Success {
		return fmt.Errorf("failed to restore project")
	}

	return nil
}

// Milestone represents a project milestone
type Milestone struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TargetDate  string `json:"targetDate,omitempty"`
	SortOrder   int    `json:"sortOrder"`
}

// MilestonesResponse is the response for listing milestones
type MilestonesResponse struct {
	Milestones []Milestone `json:"milestones"`
	Count      int         `json:"count"`
}

// GetProjectMilestones fetches milestones for a project
func (c *Client) GetProjectMilestones(ctx context.Context, projectID string) (*MilestonesResponse, error) {
	queryStr := fmt.Sprintf(`query {
		project(id: %q) {
			projectMilestones {
				nodes {
					id
					name
					description
					targetDate
					sortOrder
				}
			}
		}
	}`, projectID)

	var result struct {
		Project struct {
			ProjectMilestones struct {
				Nodes []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Description string `json:"description"`
					TargetDate  string `json:"targetDate"`
					SortOrder   int    `json:"sortOrder"`
				} `json:"nodes"`
			} `json:"projectMilestones"`
		} `json:"project"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	milestones := make([]Milestone, len(result.Project.ProjectMilestones.Nodes))
	for i, m := range result.Project.ProjectMilestones.Nodes {
		milestones[i] = Milestone{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			TargetDate:  m.TargetDate,
			SortOrder:   m.SortOrder,
		}
	}

	return &MilestonesResponse{
		Milestones: milestones,
		Count:      len(milestones),
	}, nil
}

// CreateProjectMilestone creates a new milestone for a project
func (c *Client) CreateProjectMilestone(ctx context.Context, projectID, name, description, targetDate string) (*Milestone, error) {
	inputParts := []string{
		fmt.Sprintf(`name: %q`, name),
		fmt.Sprintf(`projectId: %q`, projectID),
	}

	if description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, description))
	}
	if targetDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`targetDate: %q`, targetDate))
	}

	mutationStr := fmt.Sprintf(`mutation {
		projectMilestoneCreate(input: { %s }) {
			success
			projectMilestone {
				id
				name
				description
				targetDate
				sortOrder
			}
		}
	}`, strings.Join(inputParts, ", "))

	var result struct {
		ProjectMilestoneCreate struct {
			Success          bool `json:"success"`
			ProjectMilestone struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				TargetDate  string `json:"targetDate"`
				SortOrder   int    `json:"sortOrder"`
			} `json:"projectMilestone"`
		} `json:"projectMilestoneCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.ProjectMilestoneCreate.Success {
		return nil, fmt.Errorf("failed to create milestone")
	}

	return &Milestone{
		ID:          result.ProjectMilestoneCreate.ProjectMilestone.ID,
		Name:        result.ProjectMilestoneCreate.ProjectMilestone.Name,
		Description: result.ProjectMilestoneCreate.ProjectMilestone.Description,
		TargetDate:  result.ProjectMilestoneCreate.ProjectMilestone.TargetDate,
		SortOrder:   result.ProjectMilestoneCreate.ProjectMilestone.SortOrder,
	}, nil
}

// UpdateProjectMilestone updates a milestone
func (c *Client) UpdateProjectMilestone(ctx context.Context, milestoneID string, name, description, targetDate *string) (*Milestone, error) {
	inputParts := []string{}

	if name != nil {
		inputParts = append(inputParts, fmt.Sprintf(`name: %q`, *name))
	}
	if description != nil {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, *description))
	}
	if targetDate != nil {
		inputParts = append(inputParts, fmt.Sprintf(`targetDate: %q`, *targetDate))
	}

	if len(inputParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	mutationStr := fmt.Sprintf(`mutation {
		projectMilestoneUpdate(id: %q, input: { %s }) {
			success
			projectMilestone {
				id
				name
				description
				targetDate
				sortOrder
			}
		}
	}`, milestoneID, strings.Join(inputParts, ", "))

	var result struct {
		ProjectMilestoneUpdate struct {
			Success          bool `json:"success"`
			ProjectMilestone struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				TargetDate  string `json:"targetDate"`
				SortOrder   int    `json:"sortOrder"`
			} `json:"projectMilestone"`
		} `json:"projectMilestoneUpdate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.ProjectMilestoneUpdate.Success {
		return nil, fmt.Errorf("failed to update milestone")
	}

	return &Milestone{
		ID:          result.ProjectMilestoneUpdate.ProjectMilestone.ID,
		Name:        result.ProjectMilestoneUpdate.ProjectMilestone.Name,
		Description: result.ProjectMilestoneUpdate.ProjectMilestone.Description,
		TargetDate:  result.ProjectMilestoneUpdate.ProjectMilestone.TargetDate,
		SortOrder:   result.ProjectMilestoneUpdate.ProjectMilestone.SortOrder,
	}, nil
}

// DeleteProjectMilestone deletes a milestone
func (c *Client) DeleteProjectMilestone(ctx context.Context, milestoneID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		projectMilestoneDelete(id: %q) {
			success
		}
	}`, milestoneID)

	var result struct {
		ProjectMilestoneDelete struct {
			Success bool `json:"success"`
		} `json:"projectMilestoneDelete"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.ProjectMilestoneDelete.Success {
		return fmt.Errorf("failed to delete milestone")
	}

	return nil
}

// ProjectUpdate represents a project status update
type ProjectUpdate struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	Health    string `json:"health,omitempty"`
	CreatedAt string `json:"createdAt"`
	User      *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"user,omitempty"`
}

// ProjectUpdatesResponse is the response for listing project updates
type ProjectUpdatesResponse struct {
	Updates []ProjectUpdate `json:"updates"`
	Count   int             `json:"count"`
}

// GetProjectUpdates fetches status updates for a project
func (c *Client) GetProjectUpdates(ctx context.Context, projectID string, limit int) (*ProjectUpdatesResponse, error) {
	queryStr := fmt.Sprintf(`query {
		project(id: %q) {
			projectUpdates(first: %d) {
				nodes {
					id
					body
					health
					createdAt
					user {
						id
						displayName
					}
				}
			}
		}
	}`, projectID, limit)

	var result struct {
		Project struct {
			ProjectUpdates struct {
				Nodes []struct {
					ID        string `json:"id"`
					Body      string `json:"body"`
					Health    string `json:"health"`
					CreatedAt string `json:"createdAt"`
					User      *struct {
						ID          string `json:"id"`
						DisplayName string `json:"displayName"`
					} `json:"user"`
				} `json:"nodes"`
			} `json:"projectUpdates"`
		} `json:"project"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	updates := make([]ProjectUpdate, len(result.Project.ProjectUpdates.Nodes))
	for i, u := range result.Project.ProjectUpdates.Nodes {
		updates[i] = ProjectUpdate{
			ID:        u.ID,
			Body:      u.Body,
			Health:    u.Health,
			CreatedAt: u.CreatedAt,
		}
		if u.User != nil {
			updates[i].User = &struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			}{
				ID:          u.User.ID,
				DisplayName: u.User.DisplayName,
			}
		}
	}

	return &ProjectUpdatesResponse{
		Updates: updates,
		Count:   len(updates),
	}, nil
}

// CreateProjectUpdate creates a new status update for a project
func (c *Client) CreateProjectUpdate(ctx context.Context, projectID, body string, health *string) (*ProjectUpdate, error) {
	inputParts := []string{
		fmt.Sprintf(`projectId: %q`, projectID),
		fmt.Sprintf(`body: %q`, body),
	}

	if health != nil {
		inputParts = append(inputParts, fmt.Sprintf(`health: %s`, *health))
	}

	mutationStr := fmt.Sprintf(`mutation {
		projectUpdateCreate(input: { %s }) {
			success
			projectUpdate {
				id
				body
				health
				createdAt
				user {
					id
					displayName
				}
			}
		}
	}`, strings.Join(inputParts, ", "))

	var result struct {
		ProjectUpdateCreate struct {
			Success       bool `json:"success"`
			ProjectUpdate struct {
				ID        string `json:"id"`
				Body      string `json:"body"`
				Health    string `json:"health"`
				CreatedAt string `json:"createdAt"`
				User      *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"projectUpdate"`
		} `json:"projectUpdateCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.ProjectUpdateCreate.Success {
		return nil, fmt.Errorf("failed to create project update")
	}

	update := &ProjectUpdate{
		ID:        result.ProjectUpdateCreate.ProjectUpdate.ID,
		Body:      result.ProjectUpdateCreate.ProjectUpdate.Body,
		Health:    result.ProjectUpdateCreate.ProjectUpdate.Health,
		CreatedAt: result.ProjectUpdateCreate.ProjectUpdate.CreatedAt,
	}

	if result.ProjectUpdateCreate.ProjectUpdate.User != nil {
		update.User = &struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		}{
			ID:          result.ProjectUpdateCreate.ProjectUpdate.User.ID,
			DisplayName: result.ProjectUpdateCreate.ProjectUpdate.User.DisplayName,
		}
	}

	return update, nil
}

// DocumentListItem represents a document in a list
type DocumentListItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SlugID    string `json:"slugId"`
	Icon      string `json:"icon,omitempty"`
	URL       string `json:"url"`
	UpdatedAt string `json:"updatedAt"`
	Creator   *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"creator,omitempty"`
	Project *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project,omitempty"`
}

// DocumentsResponse is the response for listing documents
type DocumentsResponse struct {
	Documents []DocumentListItem `json:"documents"`
	Count     int                `json:"count"`
}

// DocumentSearchResponse is the response for searching documents
type DocumentSearchResponse struct {
	Documents  []DocumentListItem `json:"documents"`
	Count      int                `json:"count"`
	Query      string             `json:"query"`
	TotalCount int                `json:"totalCount"`
}

// DocumentCreateInput is the input for creating a document
type DocumentCreateInput struct {
	Title      string `json:"title"`
	Content    string `json:"content,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	TeamID     string `json:"teamId,omitempty"`
	Icon       string `json:"icon,omitempty"`
	Color      string `json:"color,omitempty"`
}

// DocumentUpdateInput is the input for updating a document
type DocumentUpdateInput struct {
	Title     string `json:"title,omitempty"`
	Content   string `json:"content,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
}

// GetDocuments fetches documents
func (c *Client) GetDocuments(ctx context.Context, projectID string, limit int) (*DocumentsResponse, error) {
	filterPart := ""
	if projectID != "" {
		filterPart = fmt.Sprintf(`, filter: { project: { id: { eq: "%s" } } }`, projectID)
	}

	queryStr := fmt.Sprintf(`query {
		documents(first: %d%s) {
			nodes {
				id
				title
				slugId
				icon
				url
				updatedAt
				creator {
					id
					displayName
				}
				project {
					id
					name
				}
			}
		}
	}`, limit, filterPart)

	var result struct {
		Documents struct {
			Nodes []struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				SlugID    string `json:"slugId"`
				Icon      string `json:"icon"`
				URL       string `json:"url"`
				UpdatedAt string `json:"updatedAt"`
				Creator   *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"creator"`
				Project *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"project"`
			} `json:"nodes"`
		} `json:"documents"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	documents := make([]DocumentListItem, len(result.Documents.Nodes))
	for i, d := range result.Documents.Nodes {
		documents[i] = DocumentListItem{
			ID:        d.ID,
			Title:     d.Title,
			SlugID:    d.SlugID,
			Icon:      d.Icon,
			URL:       d.URL,
			UpdatedAt: d.UpdatedAt,
			Creator:   d.Creator,
			Project:   d.Project,
		}
	}

	return &DocumentsResponse{
		Documents: documents,
		Count:     len(documents),
	}, nil
}

// GetDocument fetches a single document by ID
func (c *Client) GetDocument(ctx context.Context, documentID string) (*Document, error) {
	queryStr := fmt.Sprintf(`query {
		document(id: %q) {
			id
			title
			content
			icon
			color
			slugId
			url
			createdAt
			updatedAt
			creator {
				id
				displayName
			}
			project {
				id
				name
			}
		}
	}`, documentID)

	var result struct {
		Document struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Content   string `json:"content"`
			Icon      string `json:"icon"`
			Color     string `json:"color"`
			SlugID    string `json:"slugId"`
			URL       string `json:"url"`
			CreatedAt string `json:"createdAt"`
			UpdatedAt string `json:"updatedAt"`
			Creator   *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"creator"`
			Project *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"project"`
		} `json:"document"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	if result.Document.ID == "" {
		return nil, nil
	}

	return &Document{
		ID:        result.Document.ID,
		Title:     result.Document.Title,
		Content:   result.Document.Content,
		Icon:      result.Document.Icon,
		Color:     result.Document.Color,
		SlugID:    result.Document.SlugID,
		URL:       result.Document.URL,
		CreatedAt: result.Document.CreatedAt,
		UpdatedAt: result.Document.UpdatedAt,
		Creator:   result.Document.Creator,
		Project:   result.Document.Project,
	}, nil
}

// CreateDocument creates a new document
func (c *Client) CreateDocument(ctx context.Context, input DocumentCreateInput) (*Document, error) {
	inputParts := []string{
		fmt.Sprintf(`title: %q`, input.Title),
	}

	if input.Content != "" {
		inputParts = append(inputParts, fmt.Sprintf(`content: %q`, input.Content))
	}
	if input.ProjectID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`projectId: %q`, input.ProjectID))
	}
	if input.TeamID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`teamId: %q`, input.TeamID))
	}
	if input.Icon != "" {
		inputParts = append(inputParts, fmt.Sprintf(`icon: %q`, input.Icon))
	}
	if input.Color != "" {
		inputParts = append(inputParts, fmt.Sprintf(`color: %q`, input.Color))
	}

	mutationStr := fmt.Sprintf(`mutation {
		documentCreate(input: { %s }) {
			success
			document {
				id
				title
				content
				slugId
				url
				createdAt
				updatedAt
				creator {
					id
					displayName
				}
				project {
					id
					name
				}
			}
		}
	}`, strings.Join(inputParts, ", "))

	var result struct {
		DocumentCreate struct {
			Success  bool `json:"success"`
			Document struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Content   string `json:"content"`
				SlugID    string `json:"slugId"`
				URL       string `json:"url"`
				CreatedAt string `json:"createdAt"`
				UpdatedAt string `json:"updatedAt"`
				Creator   *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"creator"`
				Project *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"project"`
			} `json:"document"`
		} `json:"documentCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.DocumentCreate.Success {
		return nil, fmt.Errorf("failed to create document")
	}

	return &Document{
		ID:        result.DocumentCreate.Document.ID,
		Title:     result.DocumentCreate.Document.Title,
		Content:   result.DocumentCreate.Document.Content,
		SlugID:    result.DocumentCreate.Document.SlugID,
		URL:       result.DocumentCreate.Document.URL,
		CreatedAt: result.DocumentCreate.Document.CreatedAt,
		UpdatedAt: result.DocumentCreate.Document.UpdatedAt,
		Creator:   result.DocumentCreate.Document.Creator,
		Project:   result.DocumentCreate.Document.Project,
	}, nil
}

// UpdateDocument updates a document
func (c *Client) UpdateDocument(ctx context.Context, documentID string, input DocumentUpdateInput) (*Document, error) {
	inputParts := []string{}

	if input.Title != "" {
		inputParts = append(inputParts, fmt.Sprintf(`title: %q`, input.Title))
	}
	if input.Content != "" {
		inputParts = append(inputParts, fmt.Sprintf(`content: %q`, input.Content))
	}
	if input.ProjectID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`projectId: %q`, input.ProjectID))
	}
	if input.Icon != "" {
		inputParts = append(inputParts, fmt.Sprintf(`icon: %q`, input.Icon))
	}
	if input.Color != "" {
		inputParts = append(inputParts, fmt.Sprintf(`color: %q`, input.Color))
	}

	if len(inputParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	mutationStr := fmt.Sprintf(`mutation {
		documentUpdate(id: %q, input: { %s }) {
			success
			document {
				id
				title
				content
				slugId
				url
				createdAt
				updatedAt
				creator {
					id
					displayName
				}
				project {
					id
					name
				}
			}
		}
	}`, documentID, strings.Join(inputParts, ", "))

	var result struct {
		DocumentUpdate struct {
			Success  bool `json:"success"`
			Document struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Content   string `json:"content"`
				SlugID    string `json:"slugId"`
				URL       string `json:"url"`
				CreatedAt string `json:"createdAt"`
				UpdatedAt string `json:"updatedAt"`
				Creator   *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"creator"`
				Project *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"project"`
			} `json:"document"`
		} `json:"documentUpdate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.DocumentUpdate.Success {
		return nil, fmt.Errorf("failed to update document")
	}

	return &Document{
		ID:        result.DocumentUpdate.Document.ID,
		Title:     result.DocumentUpdate.Document.Title,
		Content:   result.DocumentUpdate.Document.Content,
		SlugID:    result.DocumentUpdate.Document.SlugID,
		URL:       result.DocumentUpdate.Document.URL,
		CreatedAt: result.DocumentUpdate.Document.CreatedAt,
		UpdatedAt: result.DocumentUpdate.Document.UpdatedAt,
		Creator:   result.DocumentUpdate.Document.Creator,
		Project:   result.DocumentUpdate.Document.Project,
	}, nil
}

// DeleteDocument archives a document
func (c *Client) DeleteDocument(ctx context.Context, documentID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		documentDelete(id: %q) {
			success
		}
	}`, documentID)

	var result struct {
		DocumentDelete struct {
			Success bool `json:"success"`
		} `json:"documentDelete"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.DocumentDelete.Success {
		return fmt.Errorf("failed to delete document")
	}

	return nil
}

// RestoreDocument restores (unarchives) a deleted document
func (c *Client) RestoreDocument(ctx context.Context, documentID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		documentUnarchive(id: %q) {
			success
		}
	}`, documentID)

	var result struct {
		DocumentUnarchive struct {
			Success bool `json:"success"`
		} `json:"documentUnarchive"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.DocumentUnarchive.Success {
		return fmt.Errorf("failed to restore document")
	}

	return nil
}

// SearchDocuments searches for documents
func (c *Client) SearchDocuments(ctx context.Context, query string, limit int) (*DocumentSearchResponse, error) {
	queryStr := fmt.Sprintf(`query {
		searchDocuments(term: %q, first: %d) {
			nodes {
				id
				title
				slugId
				icon
				url
				updatedAt
				creator {
					id
					displayName
				}
				project {
					id
					name
				}
			}
			totalCount
		}
	}`, query, limit)

	var result struct {
		SearchDocuments struct {
			Nodes []struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				SlugID    string `json:"slugId"`
				Icon      string `json:"icon"`
				URL       string `json:"url"`
				UpdatedAt string `json:"updatedAt"`
				Creator   *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"creator"`
				Project *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"project"`
			} `json:"nodes"`
			TotalCount int `json:"totalCount"`
		} `json:"searchDocuments"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	documents := make([]DocumentListItem, len(result.SearchDocuments.Nodes))
	for i, d := range result.SearchDocuments.Nodes {
		documents[i] = DocumentListItem{
			ID:        d.ID,
			Title:     d.Title,
			SlugID:    d.SlugID,
			Icon:      d.Icon,
			URL:       d.URL,
			UpdatedAt: d.UpdatedAt,
			Creator:   d.Creator,
			Project:   d.Project,
		}
	}

	return &DocumentSearchResponse{
		Documents:  documents,
		Count:      len(documents),
		Query:      query,
		TotalCount: result.SearchDocuments.TotalCount,
	}, nil
}

// Initiative represents a Linear initiative
type Initiative struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	Status      string `json:"status"`
	SlugID      string `json:"slugId"`
	URL         string `json:"url"`
	TargetDate  string `json:"targetDate,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Owner       *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"owner,omitempty"`
	Projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"projects,omitempty"`
}

// InitiativeListItem represents an initiative in a list
type InitiativeListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	SlugID      string `json:"slugId"`
	URL         string `json:"url"`
	TargetDate  string `json:"targetDate,omitempty"`
	UpdatedAt   string `json:"updatedAt"`
	Owner       *struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"owner,omitempty"`
	ProjectCount int `json:"projectCount"`
}

// InitiativesResponse is the response for listing initiatives
type InitiativesResponse struct {
	Initiatives []InitiativeListItem `json:"initiatives"`
	Count       int                  `json:"count"`
}

// InitiativeCreateInput is the input for creating an initiative
type InitiativeCreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	Status      string `json:"status,omitempty"`
	OwnerID     string `json:"ownerId,omitempty"`
	TargetDate  string `json:"targetDate,omitempty"`
}

// InitiativeUpdateInput is the input for updating an initiative
type InitiativeUpdateInput struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	Status      string `json:"status,omitempty"`
	OwnerID     string `json:"ownerId,omitempty"`
	TargetDate  string `json:"targetDate,omitempty"`
}

// GetInitiatives fetches initiatives
func (c *Client) GetInitiatives(ctx context.Context, status string, ownerID string, limit int) (*InitiativesResponse, error) {
	filterParts := []string{}
	if status != "" {
		filterParts = append(filterParts, fmt.Sprintf(`status: { eq: %q }`, status))
	}
	if ownerID != "" {
		filterParts = append(filterParts, fmt.Sprintf(`owner: { id: { eq: %q } }`, ownerID))
	}

	filterPart := ""
	if len(filterParts) > 0 {
		filterPart = fmt.Sprintf(`, filter: { %s }`, strings.Join(filterParts, ", "))
	}

	queryStr := fmt.Sprintf(`query {
		initiatives(first: %d%s) {
			nodes {
				id
				name
				status
				slugId
				targetDate
				updatedAt
				owner {
					id
					displayName
				}
				projects {
					nodes {
						id
					}
				}
			}
		}
	}`, limit, filterPart)

	var result struct {
		Initiatives struct {
			Nodes []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Status     string `json:"status"`
				SlugID     string `json:"slugId"`
				TargetDate string `json:"targetDate"`
				UpdatedAt  string `json:"updatedAt"`
				Owner      *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"owner"`
				Projects struct {
					Nodes []struct {
						ID string `json:"id"`
					} `json:"nodes"`
				} `json:"projects"`
			} `json:"nodes"`
		} `json:"initiatives"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	initiatives := make([]InitiativeListItem, len(result.Initiatives.Nodes))
	for i, init := range result.Initiatives.Nodes {
		initiatives[i] = InitiativeListItem{
			ID:           init.ID,
			Name:         init.Name,
			Status:       init.Status,
			SlugID:       init.SlugID,
			TargetDate:   init.TargetDate,
			UpdatedAt:    init.UpdatedAt,
			Owner:        init.Owner,
			ProjectCount: len(init.Projects.Nodes),
		}
	}

	return &InitiativesResponse{
		Initiatives: initiatives,
		Count:       len(initiatives),
	}, nil
}

// GetInitiative fetches a single initiative by ID
func (c *Client) GetInitiative(ctx context.Context, initiativeID string) (*Initiative, error) {
	queryStr := fmt.Sprintf(`query {
		initiative(id: %q) {
			id
			name
			description
			content
			status
			slugId
			targetDate
			createdAt
			updatedAt
			owner {
				id
				displayName
			}
			projects {
				nodes {
					id
					name
				}
			}
		}
	}`, initiativeID)

	var result struct {
		Initiative *struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Content     string `json:"content"`
			Status      string `json:"status"`
			SlugID      string `json:"slugId"`
			TargetDate  string `json:"targetDate"`
			CreatedAt   string `json:"createdAt"`
			UpdatedAt   string `json:"updatedAt"`
			Owner       *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"owner"`
			Projects struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"projects"`
		} `json:"initiative"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &result, nil); err != nil {
		return nil, err
	}

	if result.Initiative == nil {
		return nil, nil
	}

	return &Initiative{
		ID:          result.Initiative.ID,
		Name:        result.Initiative.Name,
		Description: result.Initiative.Description,
		Content:     result.Initiative.Content,
		Status:      result.Initiative.Status,
		SlugID:      result.Initiative.SlugID,
		TargetDate:  result.Initiative.TargetDate,
		CreatedAt:   result.Initiative.CreatedAt,
		UpdatedAt:   result.Initiative.UpdatedAt,
		Owner:       result.Initiative.Owner,
		Projects:    result.Initiative.Projects.Nodes,
	}, nil
}

// CreateInitiative creates a new initiative
func (c *Client) CreateInitiative(ctx context.Context, input InitiativeCreateInput) (*Initiative, error) {
	inputParts := []string{
		fmt.Sprintf(`name: %q`, input.Name),
	}

	if input.Description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, input.Description))
	}
	if input.Content != "" {
		inputParts = append(inputParts, fmt.Sprintf(`content: %q`, input.Content))
	}
	if input.Status != "" {
		inputParts = append(inputParts, fmt.Sprintf(`status: %s`, input.Status))
	}
	if input.OwnerID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`ownerId: %q`, input.OwnerID))
	}
	if input.TargetDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`targetDate: %q`, input.TargetDate))
	}

	mutationStr := fmt.Sprintf(`mutation {
		initiativeCreate(input: { %s }) {
			success
			initiative {
				id
				name
				description
				content
				status
				slugId
				targetDate
				createdAt
				updatedAt
				owner {
					id
					displayName
				}
			}
		}
	}`, strings.Join(inputParts, ", "))

	var result struct {
		InitiativeCreate struct {
			Success    bool `json:"success"`
			Initiative struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Content     string `json:"content"`
				Status      string `json:"status"`
				SlugID      string `json:"slugId"`
				TargetDate  string `json:"targetDate"`
				CreatedAt   string `json:"createdAt"`
				UpdatedAt   string `json:"updatedAt"`
				Owner       *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"owner"`
			} `json:"initiative"`
		} `json:"initiativeCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.InitiativeCreate.Success {
		return nil, fmt.Errorf("failed to create initiative")
	}

	return &Initiative{
		ID:          result.InitiativeCreate.Initiative.ID,
		Name:        result.InitiativeCreate.Initiative.Name,
		Description: result.InitiativeCreate.Initiative.Description,
		Content:     result.InitiativeCreate.Initiative.Content,
		Status:      result.InitiativeCreate.Initiative.Status,
		SlugID:      result.InitiativeCreate.Initiative.SlugID,
		TargetDate:  result.InitiativeCreate.Initiative.TargetDate,
		CreatedAt:   result.InitiativeCreate.Initiative.CreatedAt,
		UpdatedAt:   result.InitiativeCreate.Initiative.UpdatedAt,
		Owner:       result.InitiativeCreate.Initiative.Owner,
	}, nil
}

// UpdateInitiative updates an existing initiative
func (c *Client) UpdateInitiative(ctx context.Context, initiativeID string, input InitiativeUpdateInput) (*Initiative, error) {
	inputParts := []string{}

	if input.Name != "" {
		inputParts = append(inputParts, fmt.Sprintf(`name: %q`, input.Name))
	}
	if input.Description != "" {
		inputParts = append(inputParts, fmt.Sprintf(`description: %q`, input.Description))
	}
	if input.Content != "" {
		inputParts = append(inputParts, fmt.Sprintf(`content: %q`, input.Content))
	}
	if input.Status != "" {
		inputParts = append(inputParts, fmt.Sprintf(`status: %s`, input.Status))
	}
	if input.OwnerID != "" {
		inputParts = append(inputParts, fmt.Sprintf(`ownerId: %q`, input.OwnerID))
	}
	if input.TargetDate != "" {
		inputParts = append(inputParts, fmt.Sprintf(`targetDate: %q`, input.TargetDate))
	}

	if len(inputParts) == 0 {
		return nil, fmt.Errorf("at least one field must be specified to update")
	}

	mutationStr := fmt.Sprintf(`mutation {
		initiativeUpdate(id: %q, input: { %s }) {
			success
			initiative {
				id
				name
				description
				content
				status
				slugId
				targetDate
				createdAt
				updatedAt
				owner {
					id
					displayName
				}
			}
		}
	}`, initiativeID, strings.Join(inputParts, ", "))

	var result struct {
		InitiativeUpdate struct {
			Success    bool `json:"success"`
			Initiative struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Content     string `json:"content"`
				Status      string `json:"status"`
				SlugID      string `json:"slugId"`
				TargetDate  string `json:"targetDate"`
				CreatedAt   string `json:"createdAt"`
				UpdatedAt   string `json:"updatedAt"`
				Owner       *struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"owner"`
			} `json:"initiative"`
		} `json:"initiativeUpdate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return nil, err
	}

	if !result.InitiativeUpdate.Success {
		return nil, fmt.Errorf("failed to update initiative")
	}

	return &Initiative{
		ID:          result.InitiativeUpdate.Initiative.ID,
		Name:        result.InitiativeUpdate.Initiative.Name,
		Description: result.InitiativeUpdate.Initiative.Description,
		Content:     result.InitiativeUpdate.Initiative.Content,
		Status:      result.InitiativeUpdate.Initiative.Status,
		SlugID:      result.InitiativeUpdate.Initiative.SlugID,
		TargetDate:  result.InitiativeUpdate.Initiative.TargetDate,
		CreatedAt:   result.InitiativeUpdate.Initiative.CreatedAt,
		UpdatedAt:   result.InitiativeUpdate.Initiative.UpdatedAt,
		Owner:       result.InitiativeUpdate.Initiative.Owner,
	}, nil
}

// ArchiveInitiative archives an initiative
func (c *Client) ArchiveInitiative(ctx context.Context, initiativeID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		initiativeArchive(id: %q) {
			success
		}
	}`, initiativeID)

	var result struct {
		InitiativeArchive struct {
			Success bool `json:"success"`
		} `json:"initiativeArchive"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.InitiativeArchive.Success {
		return fmt.Errorf("failed to archive initiative")
	}

	return nil
}

// RestoreInitiative restores an archived initiative
func (c *Client) RestoreInitiative(ctx context.Context, initiativeID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		initiativeUnarchive(id: %q) {
			success
		}
	}`, initiativeID)

	var result struct {
		InitiativeUnarchive struct {
			Success bool `json:"success"`
		} `json:"initiativeUnarchive"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.InitiativeUnarchive.Success {
		return fmt.Errorf("failed to restore initiative")
	}

	return nil
}

// AddProjectToInitiative adds a project to an initiative
func (c *Client) AddProjectToInitiative(ctx context.Context, initiativeID, projectID string) error {
	mutationStr := fmt.Sprintf(`mutation {
		initiativeToProjectCreate(input: { initiativeId: %q, projectId: %q }) {
			success
		}
	}`, initiativeID, projectID)

	var result struct {
		InitiativeToProjectCreate struct {
			Success bool `json:"success"`
		} `json:"initiativeToProjectCreate"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.InitiativeToProjectCreate.Success {
		return fmt.Errorf("failed to add project to initiative")
	}

	return nil
}

// RemoveProjectFromInitiative removes a project from an initiative
func (c *Client) RemoveProjectFromInitiative(ctx context.Context, initiativeID, projectID string) error {
	// Query all initiativeToProject links
	queryStr := `query {
		initiativeToProjects {
			nodes {
				id
				initiative {
					id
				}
				project {
					id
				}
			}
		}
	}`

	var queryResult struct {
		InitiativeToProjects struct {
			Nodes []struct {
				ID         string `json:"id"`
				Initiative struct {
					ID string `json:"id"`
				} `json:"initiative"`
				Project struct {
					ID string `json:"id"`
				} `json:"project"`
			} `json:"nodes"`
		} `json:"initiativeToProjects"`
	}

	if err := c.graphql.Exec(ctx, queryStr, &queryResult, nil); err != nil {
		return err
	}

	// Find the link ID for the specified initiative and project
	var linkID string
	for _, link := range queryResult.InitiativeToProjects.Nodes {
		if link.Initiative.ID == initiativeID && link.Project.ID == projectID {
			linkID = link.ID
			break
		}
	}

	if linkID == "" {
		return fmt.Errorf("project not found in initiative")
	}

	// Delete the link
	mutationStr := fmt.Sprintf(`mutation {
		initiativeToProjectDelete(id: %q) {
			success
		}
	}`, linkID)

	var result struct {
		InitiativeToProjectDelete struct {
			Success bool `json:"success"`
		} `json:"initiativeToProjectDelete"`
	}

	if err := c.graphql.Exec(ctx, mutationStr, &result, nil); err != nil {
		return err
	}

	if !result.InitiativeToProjectDelete.Success {
		return fmt.Errorf("failed to remove project from initiative")
	}

	return nil
}
