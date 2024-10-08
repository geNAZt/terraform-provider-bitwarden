package bw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type LoggingRoundTripper struct {
	ctx context.Context
}

func (t LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	tflog.Debug(t.ctx, "Request", map[string]any{"method": req.Method, "url": req.URL.String()})

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	tflog.Debug(t.ctx, "Response", map[string]any{"status": resp.Status})
	return resp, err
}

type restClient struct {
	ctx          context.Context
	retryHandler *retryHandler

	client   *http.Client
	endpoint string
}

func retry[T any](r *restClient, fn func() (T, error)) (T, error) {
	attempts := 0
	for {
		attempts = attempts + 1
		out, err := fn()
		if err == nil || !r.retryHandler.IsRetryable(err, attempts) {
			return out, err
		}

		r.retryHandler.Backoff(attempts)
		log.Printf("[ERROR] Retrying command after error: %v\n", err)
	}
}

func (r *restClient) CreateAttachment(itemId, filePath string) (*Object, error) {
	tflog.Debug(r.ctx, "Creating attachment", map[string]any{"itemId": itemId})

	return retry(r, func() (*Object, error) {
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

		o, sErr := readResponse[Object](r.ctx, resp)
		if len(sErr) > 0 {
			return nil, fmt.Errorf(sErr)
		}

		return o, nil
	})
}

func (r *restClient) CreateObject(object Object) (*Object, error) {
	tflog.Debug(r.ctx, "Creating object", map[string]any{"itemId": object.ID})

	return retry(r, func() (*Object, error) {
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

		o, sErr := readResponse[Object](r.ctx, resp)
		if len(sErr) > 0 {
			return nil, fmt.Errorf(sErr)
		}

		return o, nil
	})
}

func (r *restClient) EditObject(object Object) (*Object, error) {
	tflog.Debug(r.ctx, "Editing object", map[string]any{"itemId": object.ID})

	return retry(r, func() (*Object, error) {
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

		o, sErr := readResponse[Object](r.ctx, resp)
		if len(sErr) > 0 {
			return nil, fmt.Errorf(sErr)
		}

		return o, nil
	})
}

func (r *restClient) GetAttachment(itemId, attachmentId string) ([]byte, error) {
	tflog.Debug(r.ctx, "Getting attachement", map[string]any{"itemId": itemId, "attachement": attachmentId})

	return retry(r, func() ([]byte, error) {
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

		if resp.StatusCode == 404 {
			return nil, ErrAttachmentNotFound
		}

		all, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return all, nil
	})
}

func (r *restClient) GetObject(object Object) (*Object, error) {
	tflog.Debug(r.ctx, "Getting object", map[string]any{"itemId": object.ID})

	return retry(r, func() (*Object, error) {
		u, err := url.Parse(r.endpoint)
		if err != nil {
			return nil, err
		}

		u = u.JoinPath("object", "item", object.ID)
		resp, err := r.client.Get(u.String())
		if err != nil {
			return nil, err
		}

		o, sErr := readResponse[Object](r.ctx, resp)
		if len(sErr) > 0 {
			if sErr == "Not found." {
				return nil, ErrObjectNotFound
			}

			return nil, fmt.Errorf(sErr)
		}

		return o, nil
	})
}

func (r *restClient) GetSessionKey() string {
	return "" // REST clients don't need this
}

func (r *restClient) ListObjects(objType string, options ...ListObjectsOption) ([]Object, error) {
	tflog.Debug(r.ctx, "List objects", map[string]any{"type": objType})

	return retry(r, func() ([]Object, error) {
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

		l, sErr := readArrayResponse[Object](r.ctx, resp)
		if len(sErr) > 0 {
			return nil, fmt.Errorf(sErr)
		}

		return l, nil
	})
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

	return retry(r, func() (*Status, error) {
		u, err := url.Parse(r.endpoint)
		if err != nil {
			return nil, err
		}

		u = u.JoinPath("status")
		resp, err := r.client.Get(u.String())
		if err != nil {
			return nil, err
		}

		re, sErr := readResponse[RESTStatus](r.ctx, resp)
		if len(sErr) > 0 {
			return nil, fmt.Errorf(sErr)
		}

		return &re.Template, nil
	})
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

	_, sErr := readResponse[RESTStatus](r.ctx, resp)
	if len(sErr) > 0 {
		return fmt.Errorf(sErr)
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

	_, sErr := readResponse[RESTMessageResult](r.ctx, resp)
	if len(sErr) > 0 {
		return fmt.Errorf(sErr)
	}

	return nil
}

func readResponse[T any](ctx context.Context, resp *http.Response) (*T, string) {
	var respObj RESTWrapper[T]
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err.Error()
	}

	tflog.Debug(ctx, "Response from BW", map[string]any{"raw": string(respData)})

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return nil, err.Error()
	}

	if respObj.Success {
		return &respObj.Data, ""
	}

	return nil, respObj.Message
}

func readArrayResponse[T any](ctx context.Context, resp *http.Response) ([]T, string) {
	var respObj RESTWrapper[[]T]
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err.Error()
	}

	tflog.Debug(ctx, "Response from BW", map[string]any{"raw": string(respData)})

	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return nil, err.Error()
	}

	if respObj.Success {
		return respObj.Data, ""
	}

	return nil, respObj.Message
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
	rt := LoggingRoundTripper{ctx: ctx}

	return &restClient{
		ctx:          ctx,
		retryHandler: &retryHandler{},

		client:   &http.Client{Transport: rt},
		endpoint: endpoint,
	}
}
