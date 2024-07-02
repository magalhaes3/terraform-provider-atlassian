package atlassian

import (
	"context"
	"fmt"
	"strconv"

	jira "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type (
	jiraIssueScreenDataSource struct {
		p atlassianProvider
	}
	jiraIssueScreenDataSourceModel struct {
		ID          types.String `tfsdk:"id"`
		Name        types.String `tfsdk:"name"`
		Description types.String `tfsdk:"description"`
	}
)

var (
	_ datasource.DataSource = (*jiraIssueScreenDataSource)(nil)
)

func NewJiraIssueScreenDataSource() datasource.DataSource {
	return &jiraIssueScreenDataSource{}
}

func (*jiraIssueScreenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jira_issue_screen"
}

func (*jiraIssueScreenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Jira Issue Screen Data Source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the issue screen.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the screen." +
					"The name must be unique." +
					"The maximum length is 255 characters.",
				Computed: true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the screen." +
					"The maximum length is 255 characters.",
				Computed: true,
			},
		},
	}
}

func (d *jiraIssueScreenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *jiraIssueScreenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading issue screen data source")

	var newState jiraIssueScreenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded issue screen config", map[string]interface{}{
		"readConfig": fmt.Sprintf("%+v", newState),
	})

	issueScreenId, err := strconv.Atoi(newState.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Unable to parse value of \"id\" attribute.", "Value of \"id\" attribute can only be a numeric string.")
		return
	}

	screenParamsScheme := models.ScreenParamsScheme{
		IDs: []int{issueScreenId},
	}

	issueScreen, res, err := d.p.jira.Screen.Gets(ctx, &screenParamsScheme, 0, 50)
	if err != nil {
		var resBody string
		if res != nil {
			resBody = res.Bytes.String()
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get issue screen, got error: %s\n%s", err.Error(), resBody))
		return
	}
	tflog.Debug(ctx, "Retrieve issue screen from API state", map[string]interface{}{
		"readApiState": fmt.Sprintf("%+v", issueScreen),
	})

	newState.Name = types.StringValue(issueScreen.Values[0].Name)
	newState.Description = types.StringValue(issueScreen.Values[0].Description)

	tflog.Debug(ctx, "Storing issue screen info into the state")
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
