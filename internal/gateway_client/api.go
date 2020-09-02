package gateway_client

import (
	"errors"
	"fmt"
	"net/http"

	v1 "github.com/TykTechnologies/tyk-operator/api/v1"
	"github.com/levigross/grequests"
)

var (
	apiCollisionError = errors.New("api id, listen path or slug collision")
)

type Api struct {
	*Client
}

func (a Api) All() ([]v1.APIDefinitionSpec, error) {
	fullPath := JoinUrl(a.url, endpointAPIs)

	res, err := grequests.Get(fullPath, a.opts)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API Returned error: %d", res.StatusCode)
	}

	var list []v1.APIDefinitionSpec
	if err := res.JSON(&list); err != nil {
		return nil, err
	}

	return list, nil
}

func (a Api) Create(def *v1.APIDefinitionSpec) (string, error) {
	// get all apis
	list, err := a.All()
	if err != nil {
		return "", err
	}

	// check exists / collisions
	for _, api := range list {
		if api.APIID == def.APIID {
			return "", apiCollisionError
		}

		if api.Proxy.ListenPath == def.Proxy.ListenPath {
			return "", apiCollisionError
		}

		if api.Slug == def.Slug {
			return "", apiCollisionError
		}
	}

	// Create
	opts := a.opts
	opts.JSON = def
	fullPath := JoinUrl(a.url, endpointAPIs)

	res, err := grequests.Post(fullPath, opts)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API Returned error: %v (code: %v)", res.String(), res.StatusCode)
	}

	var resMsg ResponseMsg
	if err := res.JSON(&resMsg); err != nil {
		return "", err
	}

	if resMsg.Status != "ok" {
		return "", fmt.Errorf("API request completed, but with error: %s", resMsg.Message)
	}

	return resMsg.Key, nil
}

func (a Api) Update(def *v1.APIDefinitionSpec) error {
	list, err := a.All()
	if err != nil {
		return err
	}

	var apiToUpdate *v1.APIDefinitionSpec
	for _, api := range list {
		if api.APIID == def.APIID {
			apiToUpdate = &api
			break
		}
	}

	if apiToUpdate == nil {
		return notFoundError
	}

	// Update
	opts := a.opts
	opts.JSON = def
	fullPath := JoinUrl(a.url, endpointAPIs, apiToUpdate.APIID)

	res, err := grequests.Put(fullPath, opts)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("API Returned error: %v (code: %v)", res.String(), res.StatusCode)
	}

	var resMsg ResponseMsg
	if err := res.JSON(&resMsg); err != nil {
		return err
	}

	if resMsg.Status != "ok" {
		return fmt.Errorf("API request completed, but with error: %s", resMsg.Message)
	}

	return nil
}

func (a Api) Delete(id string) error {
	delPath := JoinUrl(a.url, endpointAPIs, id)

	res, err := grequests.Delete(delPath, a.opts)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("API Returned error: %s", res.String())
	}

	return nil
}