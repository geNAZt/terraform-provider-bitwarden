package bw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type restClient struct {
	client   *http.Client
	endpoint string
}

func (r *restClient) CreateAttachment(itemId, filePath string) (*Object, error) {
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

	return readResponse[Object](resp)
}

func (r *restClient) CreateObject(object Object) (*Object, error) {
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

	return readResponse[Object](resp)
}

func (r *restClient) EditObject(object Object) (*Object, error) {
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

	return readResponse[Object](resp)
}

func (r *restClient) GetAttachment(itemId, attachmentId string) ([]byte, error) {
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
	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("object", "item", object.ID)
	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	return readResponse[Object](resp)
}

func (r *restClient) GetSessionKey() string {
	return "" // REST clients don't need this
}

func (r *restClient) ListObjects(objType string, options ...ListObjectsOption) ([]Object, error) {
	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("list", "object", "items")
	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	return readArrayResponse[Object](resp)
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
	u, err := url.Parse(r.endpoint)
	if err != nil {
		return nil, err
	}

	u = u.JoinPath("status")
	resp, err := r.client.Get(u.String())
	if err != nil {
		return nil, err
	}

	re, err := readResponse[RESTStatus](resp)
	if err != nil {
		return nil, err
	}

	return &re.Template, nil
}

func (r *restClient) Sync() error {
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

	_, err = readResponse[RESTStatus](resp)
	if err != nil {
		return err
	}

	return nil
}

func (r *restClient) Unlock(password string) error {
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

	_, err = readResponse[RESTMessageResult](resp)
	if err != nil {
		return err
	}

	return nil
}

func readResponse[T any](resp *http.Response) (*T, error) {
	var respObj RESTWrapper[T]
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return nil, err
	}

	if respObj.Success {
		return &respObj.Data, nil
	}

	return nil, fmt.Errorf("response was not successful")
}

func readArrayResponse[T any](resp *http.Response) ([]T, error) {
	var respObj RESTWrapper[[]T]
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

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

func NewRestClient(endpoint string) Client {
	return &restClient{
		client:   http.DefaultClient,
		endpoint: endpoint,
	}
}
