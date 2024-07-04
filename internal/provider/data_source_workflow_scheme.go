package atlassian

import (
	"context"
	"fmt"
	"strconv"

	jira "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type (
	jiraWorkflowSchemeDataSource struct {
		p atlassianProvider
	}
	jiraWorkflowSchemeDataSourceModel struct {
		ID          types.String `tfsdk:"id"`
		Name        types.String `tfsdk:"name"`
		Description types.String `tfsdk:"description"`
	}
)

var (
	_ datasource.DataSource = (*jiraWorkflowSchemeDataSource)(nil)
)

func NewJiraWorkflowSchemeDataSource() datasource.DataSource {
	return &jiraWorkflowSchemeDataSource{}
}

func (*jiraWorkflowSchemeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jira_workflow_scheme"
}

func (*jiraWorkflowSchemeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Jira Workflow Scheme Data Source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the workflow scheme.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the workflow scheme.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the workflow scheme.",
				Computed:            true,
			},
		},
	}
}

func (d *jiraWorkflowSchemeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*jira.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *jira.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.p.jira = client
}

func (d *jiraWorkflowSchemeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading workflow scheme data source")

	var newState jiraWorkflowSchemeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded status config", map[string]interface{}{
		"readConfig": fmt.Sprintf("%+v", newState),
	})

	workflowSchemeId, err := strconv.Atoi(newState.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Unable to parse value of \"id\" attribute.", "Value of \"id\" attribute can only be a numeric string.")
		return
	}

	workflowScheme, res, err := d.p.jira.Workflow.Scheme.Get(ctx, workflowSchemeId, false)
	if err != nil {
		var resBody string
		if res != nil {
			resBody = res.Bytes.String()
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Jira workflow scheme, got error: %s\n%s", err.Error(), resBody))
		return
	}
	tflog.Debug(ctx, "Retrieve status from API state", map[string]interface{}{
		"readApiState": fmt.Sprintf("%+v", workflowScheme),
	})

	newState.Name = types.StringValue(workflowScheme.Name)
	newState.Description = types.StringValue(workflowScheme.Description)

	tflog.Debug(ctx, "Storing workflow scheme info into the state")
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
