```go
package task

type TaskController struct {
    log *slog.Logger
    authorizer authz.Authorizer
    service tasks.TaskCreator
}

type CreateTaskRequest struct {
    Name string
    Description string
}

type TaskCreatedSchema struct {
    ID string
}

// @Responder
type CreateTaskResponder interface {
	// @Status 403
	Forbidden()
	// @Status 422
	Unprocessable()
	// @Status 200
    Created(created TaskCreatedSchema)

}

// @Resource POST /api/tasks/public/v1/scope/{scope}/
// @Param organizationID Header X-Tasks-Organization-Id
// @Summary create task with specified scope
// @Details Create task with specified scope
// @Details Returns id of new created task
// @Tag task
// @Tag common
func (c *TaskController) CreateTask(ctx context.Context, r CreateTaskResponder, organizationID string, scope string, verbose bool, body CreateTaskRequest) error {
    if err := c.authorizer.HasPermission(ctx, organizationID, "tasks:create"); err != nil {
        r.Forbidden()
        return nil
    }

    if scope == "" {
        r.Unprocessable()
        return nil
    }

    id, err := c.service.Create(ctx, organizationID, tasks.Task{
        Name: body.Name,
        Description: body.Description,
    })

    if err != nil {
        return err
    }

	r.Created(CreateTaskResponses{
        Status: http.StatusOk,
        Body: TaskCreatedSchema{
            ID: id,
        },
    })

	return nil
}
```