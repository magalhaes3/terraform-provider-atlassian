package atlassian

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	jira "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/openscientia/terraform-provider-atlassian/internal/provider/planmodifiers/stringmodifiers"
)

type (
	jiraProjectResource struct {
		p atlassianProvider
	}

	jiraProjectResourceModel struct {
		ID                       types.String `tfsdk:"id"`
		Key                      types.String `tfsdk:"key"`
		Name                     types.String `tfsdk:"name"`
		Description              types.String `tfsdk:"description"`
		AvatarId                 types.Int64  `tfsdk:"avatar_id"`
		FieldConfigurationScheme types.Int64  `tfsdk:"field_configuration_scheme"`
		IssueTypeScheme          types.Int64  `tfsdk:"issue_type_scheme"`
		IssueTypeScreenScheme    types.Int64  `tfsdk:"issue_type_screen_scheme"`
		WorkflowScheme           types.Int64  `tfsdk:"workflow_scheme"`
		LeadAccountId            types.String `tfsdk:"lead_account_id"`
		ProjectTypeKey           types.String `tfsdk:"project_type_key"`
		URL                      types.String `tfsdk:"url"`
	}
)

var (
	_ resource.Resource                = (*jiraProjectResource)(nil)
	_ resource.ResourceWithImportState = (*jiraProjectResource)(nil)
)

func NewJiraProjectResource() resource.Resource {
	return &jiraProjectResource{}
}

func (*jiraProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jira_project"
}

func (*jiraProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Jira Project Resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Project keys must be unique and start with an uppercase letter followed by one or more uppercase alphanumeric characters. The maximum length is 10 characters.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(10),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the project.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A brief description of the project.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringmodifiers.DefaultValue(""),
				},
			},
			"avatar_id": schema.Int64Attribute{
				MarkdownDescription: "An integer value for the project's avatar.",
				Optional:            true,
			},
			"field_configuration_scheme": schema.Int64Attribute{
				MarkdownDescription: "The ID of the field configuration scheme for the project.",
				Optional:            true,
			},
			"issue_type_scheme": schema.Int64Attribute{
				MarkdownDescription: "The ID of the issue type scheme for the project. If you specify the issue type scheme you cannot specify the project template key.",
				Optional:            true,
			},
			"issue_type_screen_scheme": schema.Int64Attribute{
				MarkdownDescription: "The ID of the issue type screen scheme for the project. If you specify the issue type screen scheme you cannot specify the project template key.",
				Optional:            true,
			},
			"workflow_scheme": schema.Int64Attribute{
				MarkdownDescription: "The ID of the workflow scheme for the project. If you specify the workflow scheme you cannot specify the project template key.",
				Optional:            true,
			},
			"lead_account_id": schema.StringAttribute{
				MarkdownDescription: "The account ID of the project lead. Either lead or leadAccountId must be set when creating a project. Cannot be provided with lead.",
				Optional:            true,
				Computed:            true,
			},
			"project_type_key": schema.StringAttribute{
				MarkdownDescription: "The project type, which defines the application-specific feature set. If you don't specify the project template you have to specify the project type. Valid values: software, service_desk, business",
				Optional:            true,
				Computed:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "A link to information about this project, such as project documentation.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringmodifiers.DefaultValue(""),
				},
			},
		},
	}
}

func (r *jiraProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*jira.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *jira.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.p.jira = client
}

func (*jiraProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *jiraProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating project")

	var plan jiraProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded project plan", map[string]interface{}{
		"createPlan": fmt.Sprintf("%+v", plan),
	})

	projectPayload := new(models.ProjectPayloadScheme)
	projectPayload.Key = plan.Key.ValueString()
	projectPayload.Name = plan.Name.ValueString()
	projectPayload.Description = plan.Description.ValueString()
	projectPayload.AvatarID = int(plan.AvatarId.ValueInt64())
	projectPayload.FieldConfigurationScheme = int(plan.FieldConfigurationScheme.ValueInt64())
	projectPayload.IssueTypeScheme = int(plan.IssueTypeScheme.ValueInt64())
	projectPayload.IssueTypeScreenScheme = int(plan.IssueTypeScreenScheme.ValueInt64())
	projectPayload.LeadAccountID = plan.LeadAccountId.ValueString()
	projectPayload.ProjectTypeKey = plan.ProjectTypeKey.ValueString()
	projectPayload.URL = plan.URL.ValueString()
	projectPayload.WorkflowScheme = int(plan.WorkflowScheme.ValueInt64())

	returnedProject, res, err := r.p.jira.Project.Create(ctx, projectPayload)
	if err != nil {
		var resBody string
		if res != nil {
			resBody = res.Bytes.String()
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s\n%s", err, resBody))
		return
	}
	tflog.Debug(ctx, "Created project")

	plan.ID = types.StringValue(strconv.Itoa(returnedProject.ID))

	tflog.Debug(ctx, "Storing project into the state", map[string]interface{}{
		"createNewState": fmt.Sprintf("%+v", plan),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jiraProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading project resource")

	var state jiraProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded project from state", map[string]interface{}{
		"readState": fmt.Sprintf("%+v", state),
	})

	projectID := state.ID.ValueString()

	project, res, err := r.p.jira.Project.Get(ctx, projectID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get project, got error: %s\n%s", err.Error(), res.Bytes.String()))
		return
	}
	tflog.Debug(ctx, "Retrieved project from API state")

	state.ID = types.StringValue(project.ID)
	state.Key = types.StringValue(project.Key)
	state.Name = types.StringValue(project.Name)
	state.Description = types.StringValue(project.Description)
	avatarUrl, _ := url.Parse(project.AvatarUrls.One6X16)
	avatarID, _ := strconv.Atoi(strings.Split(avatarUrl.Path, "/")[9])
	state.AvatarId = types.Int64Value(int64(avatarID))
	state.LeadAccountId = types.StringValue(project.Lead.AccountID)
	state.ProjectTypeKey = types.StringValue(project.ProjectTypeKey)
	state.URL = types.StringValue(project.URL)

	projectIDInt, _ := strconv.Atoi(projectID)
	issueTypesSchemes, res, err := r.p.jira.Issue.Type.Scheme.Projects(ctx, []int{projectIDInt}, 0, 1)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get issue type schemes for project, got error: %s\n%s", err.Error(), res.Bytes.String()))
		return
	}

	for _, issueTypeScheme := range issueTypesSchemes.Values {
		tflog.Info(ctx, issueTypeScheme.IssueTypeScheme.Name)
		for issueTypeSchemeProjectId := range issueTypeScheme.ProjectIds {
			if issueTypeSchemeProjectId == projectIDInt {
				issueTypeSchemeId, _ := strconv.Atoi(issueTypeScheme.IssueTypeScheme.ID)
				state.IssueTypeScheme = types.Int64Value(int64(issueTypeSchemeId))
				break
			}
		}
	}

	tflog.Debug(ctx, "Storing issue type into the state", map[string]interface{}{
		"readNewState": fmt.Sprintf("%+v", state),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *jiraProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating project resource")

	var plan jiraProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded project plan", map[string]interface{}{
		"updatePlan": fmt.Sprintf("%+v", plan),
	})

	var state jiraProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded project from state", map[string]interface{}{
		"updateState": fmt.Sprintf("%+v", state),
	})

	projectID := state.ID.ValueString()

	projectPayload := new(models.ProjectUpdateScheme)
	projectPayload.Key = plan.Key.ValueString()
	projectPayload.Name = plan.Name.ValueString()
	projectPayload.Description = plan.Description.ValueString()
	projectPayload.AvatarID = int(plan.AvatarId.ValueInt64())
	projectPayload.ProjectTypeKey = plan.ProjectTypeKey.ValueString()
	projectPayload.URL = plan.URL.ValueString()

	returnedProject, res, err := r.p.jira.Project.Update(ctx, projectID, projectPayload)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update issue type, got error: %s\n%s", err.Error(), res.Bytes.String()))
		return
	}
	tflog.Debug(ctx, "Updated project in API state")

	response, err := r.p.jira.Issue.Type.Scheme.Assign(ctx, plan.IssueTypeScheme.String(), returnedProject.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to assign issue type scheme to project, got error: %s\n%s", err.Error(), response.Bytes.String()))
		return
	}
	tflog.Debug(ctx, "Assigned issue type scheme to project")

	avatarUrl, _ := url.Parse(returnedProject.AvatarUrls.One6X16)
	avatarID, _ := strconv.Atoi(strings.Split(avatarUrl.Path, "/")[9])

	var result = jiraProjectResourceModel{
		ID:              types.StringValue(returnedProject.ID),
		Key:             types.StringValue(returnedProject.Key),
		Name:            types.StringValue(returnedProject.Name),
		Description:     types.StringValue(returnedProject.Description),
		AvatarId:        types.Int64Value(int64(avatarID)),
		IssueTypeScheme: types.Int64Value(plan.IssueTypeScheme.ValueInt64()),
		LeadAccountId:   types.StringValue(returnedProject.Lead.AccountID),
		ProjectTypeKey:  types.StringValue(returnedProject.ProjectTypeKey),
		URL:             types.StringValue(returnedProject.URL),
	}

	tflog.Debug(ctx, "Storing issue type into the state")
	resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
}

func (r *jiraProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting project resource")

	var state jiraProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "Loaded project from state")

	res, err := r.p.jira.Project.Delete(ctx, state.ID.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s\n%s", err, res.Bytes.String()))
		return
	}
	tflog.Debug(ctx, "Deleted project from API state")

	// If a Resource type Delete method is completed without error, the framework will automatically remove the resource.
}
