package jobrunner

import (
	"errors"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
)

type ResponseWrapperMock struct {
}

func (r *ResponseWrapperMock) DoRaw() ([]byte, error) {
	return nil, errors.New("ResponseWrapperMock does not implement this")
}

func (r *ResponseWrapperMock) Stream() (io.ReadCloser, error) {
	return nil, errors.New("ResponseWrapperMock does not implement this")
}

type ServiceInterfaceMock struct {
	ListCalledWith *metav1.ListOptions
	ListError      error
	ListResponse   *v1.ServiceList
}

func (s *ServiceInterfaceMock) Create(*v1.Service) (*v1.Service, error) {
	return nil, errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) Update(*v1.Service) (*v1.Service, error) {
	return nil, errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) UpdateStatus(*v1.Service) (*v1.Service, error) {
	return nil, errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) Delete(name string, options *metav1.DeleteOptions) error {
	return errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) Get(name string, options metav1.GetOptions) (*v1.Service, error) {
	return nil, errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) List(opts metav1.ListOptions) (*v1.ServiceList, error) {
	optsCopy := opts
	s.ListCalledWith = &optsCopy

	if s.ListError != nil {
		return nil, s.ListError
	} else {
		return s.ListResponse, nil
	}
}

func (s *ServiceInterfaceMock) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Service, err error) {
	return nil, errors.New("ServiceInterfaceMock does not implement this")
}

func (s *ServiceInterfaceMock) ProxyGet(scheme, name, port, path string, params map[string]string) restclient.ResponseWrapper {
	nilMock := ResponseWrapperMock{}
	return &nilMock
}
