package bw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type restClient struct {
	ctx context.Context

	client   *http.Client
	endpoint string
}

func (r *restClient) CreateAttachment(itemId, filePath string) (*Object, error) {
	tflog.Debug(r.ctx, "Creating attachement", map[string]any{"itemId": itemId})

	// Prepare file for upload
	var (
		buf = new(bytes.Buffer)
		w   = multipart.NewWriter(buf)
	)

	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	_, err = part.Write(data)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("attachment")
	q := u.Query()
	q.Set("itemid", itemId)
	u.RawQuery = q.Encode()

	resp, err := r.client.Post(u.String(), w.FormDataContentType(), buf)
	if err != nil {
		return nil, err
	}

	return readResponse[Object](r.ctx, resp)
}

func (r *restClient) CreateObject(object Object) (*Object, error) {
	tflog.Debug(r.ctx, "Creating object", map[string]any{"itemId": object.ID})

	requestData, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("object", "item")
	resp, err := r.client.Post(u.String(), "application/json", bytes.NewBuffer(requestData))
	if err != nil {
		return nil, err
	}

	return readResponse[Object](r.ctx, resp)
}

func (r *restClient) EditObject(object Object) (*Object, error) {
	tflog.Debug(r.ctx, "Editing object", map[string]any{"itemId": object.ID})

	requestData, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("object", "item", object.ID)
	request, err := http.NewRequest("PUT", u.String(), bytes.NewBuffer(requestData))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(request)
	if err != nil {
		return nil, err
	}

	return readResponse[Object](r.ctx, resp)
}

func (r *restClient) GetAttachment(itemId, attachmentId string) ([]byte, error) {
	tflog.Debug(r.ctx, "Getting attachement", map[string]any{"itemId": itemId, "attachement": attachmentId})

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("object", "attachment", attachmentId)
	q := u.Query()
	q.Set("itemid", itemId)
	u.RawQuery = q.Encode()

	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return all, nil
}

func (r *restClient) GetObject(object Object) (*Object, error) {
	tflog.Debug(r.ctx, "Getting object", map[string]any{"itemId": object.ID})

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("object", "item", object.ID)
	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	return readResponse[Object](r.ctx, resp)
}

func (r *restClient) GetSessionKey() string {
	return "" // REST clients don't need this
}

func (r *restClient) ListObjects(objType string, options ...ListObjectsOption) ([]Object, error) {
	tflog.Debug(r.ctx, "List objects", map[string]any{"type": objType})

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("list", "object", objType)

	// Add filters
	q := u.Query()
	for _, opt := range options {
		opt(nil, &q)
	}

	u.RawQuery = q.Encode()

	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	return readArrayResponse[Object](r.ctx, resp)
}

func (r *restClient) LoginWithAPIKey(password, clientId, clientSecret string) error {
	return fmt.Errorf("rest client doesn't support login")
}

func (r *restClient) LoginWithPassword(username, password string) error {
	return fmt.Errorf("rest client doesn't support login")
}

func (r *restClient) Logout() error {
	return fmt.Errorf("rest client doesn't support logout")
}

func (r *restClient) DeleteAttachment(itemId, attachmentId string) error {
	tflog.Debug(r.ctx, "Delete attachement", map[string]any{"itemId": itemId, "attachementId": attachmentId})

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	u = u.JoinPath("object", "attachment", attachmentId)
	q := u.Query()
	q.Set("itemid", itemId)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	return readBooleanResponse(resp)
}

func (r *restClient) DeleteObject(object Object) error {
	tflog.Debug(r.ctx, "Deleting object", map[string]any{"itemId": object.ID})

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	u = u.JoinPath("object", "item", object.ID)

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	return readBooleanResponse(resp)
}

func (r *restClient) SetServer(s string) error {
	return fmt.Errorf("rest client doesn't support switching servers")
}

func (r *restClient) SetSessionKey(s string) {

}

func (r *restClient) Status() (*Status, error) {
	tflog.Debug(r.ctx, "Getting status")

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("status")
	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	re, err := readResponse[RESTStatus](r.ctx, resp)
	if err != nil {
		return nil, err
	}

	return &re.Template, nil
}

func (r *restClient) Sync() error {
	tflog.Debug(r.ctx, "Sync vault")

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	u = u.JoinPath("sync")

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	_, err = readResponse[RESTStatus](r.ctx, resp)
	if err != nil {
		return err
	}

	return nil
}

func (r *restClient) Unlock(password string) error {
	tflog.Debug(r.ctx, "Unlock vault")

	rp := &RESTUnlock{Password: password}

	requestData, err := json.Marshal(rp)
	if err != nil {
		return err
	}

	u, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}

	u = u.JoinPath("unlock")
	request, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(requestData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(request)
	if err != nil {
		return err
	}

	_, err = readResponse[RESTMessageResult](r.ctx, resp)
	if err != nil {
		return err
	}

	return nil
}

func readResponse[T any](ctx context.Context, resp *http.Response) (*T, error) {
	var respObj RESTWrapper[T]
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "Response from BW", map[string]any{"raw": string(respData)})

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return nil, err
	}

	if respObj.Success {
		return &respObj.Data, nil
	}

	return nil, fmt.Errorf("response was not successful")
}

func readArrayResponse[T any](ctx context.Context, resp *http.Response) ([]T, error) {
	var respObj RESTWrapper[[]T]
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "Response from BW", map[string]any{"raw": string(respData)})

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return nil, err
	}

	if respObj.Success {
		return respObj.Data, nil
	}

	return nil, fmt.Errorf("response was not successful")
}

func readBooleanResponse(resp *http.Response) error {
	var respObj RESTSuccess
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return err
	}

	if !respObj.Success {
		return fmt.Errorf("response was not successful")
	}

	return nil
}

func NewRestClient(ctx context.Context, endpoint string) Client {
	return &restClient{
		ctx: ctx,

		client:   http.DefaultClient,
		endpoint: endpoint,
	}
}
