package atlassian

import (
	"context"
	"fmt"

	jira "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type (
	jiraStatusDataSource struct {
		p atlassianProvider
	}
	jiraStatusDataSourceModel struct {
		ID          types.String `tfsdk:"id"`
		Name        types.String `tfsdk:"name"`
		Description types.String `tfsdk:"description"`
		Category    types.String `tfsdk:"category"`
	}
)

var (
	_ datasource.DataSource = (*jiraStatusDataSource)(nil)
)

func NewJiraStatusDataSource() datasource.DataSource {
	return &jiraStatusDataSource{}
}

func (*jiraStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jira_status"
}

func (*jiraStatusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Jira Status Data Source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the status.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the status." +
					"The name must be unique." +
					"The maximum length is 255 characters.",
				Computed: true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the status." +
					"The maximum length is 255 characters.",
				Computed: true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "The category of the status.",
				Computed:            true,
			},
		},
	}
}

func (d *jiraStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *jiraStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading status data source")

	var newState jiraStatusDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded status config", map[string]interface{}{
		"readConfig": fmt.Sprintf("%+v", newState),
	})

	statusId := newState.ID.ValueString()
	if statusId == "" {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "Unable to parse value of \"id\" attribute.", "Value of \"id\" attribute can only be a numeric string.")
		return
	}

	status, res, err := d.p.jira.Workflow.Status.Gets(ctx, []string{statusId}, nil)
	if err != nil {
		var resBody string
		if res != nil {
			resBody = res.Bytes.String()
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Jira status, got error: %s\n%s", err.Error(), resBody))
		return
	}
	tflog.Debug(ctx, "Retrieve status from API state", map[string]interface{}{
		"readApiState": fmt.Sprintf("%+v", status),
	})

	newState.Name = types.StringValue(status[0].Name)
	newState.Description = types.StringValue(status[0].Description)
	newState.Category = types.StringValue(status[0].StatusCategory)

	tflog.Debug(ctx, "Storing status info into the state")
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
