package jobrunner

import (
	"errors"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type JobInterfaceMock struct {
	CreateErr     error
	GetCalledWith *string
	GetResult     *v1.Job
	GetError      error

	ListCalledWith *metav1.ListOptions
	ListError      error
	ListResult     *v1.JobList
}

func (j *JobInterfaceMock) Create(newJob *v1.Job) (*v1.Job, error) {
	if j.CreateErr != nil {
		return nil, j.CreateErr
	} else {
		return newJob, nil
	}
}

func (j *JobInterfaceMock) Update(*v1.Job) (*v1.Job, error) {
	return nil, errors.New("JobInterfaceMock does not implement this")
}

func (j *JobInterfaceMock) UpdateStatus(*v1.Job) (*v1.Job, error) {
	return nil, errors.New("JobInterfaceMock does not implement this")
}

func (j *JobInterfaceMock) Delete(name string, options *metav1.DeleteOptions) error {
	return errors.New("JobInterfaceMock does not implement this")
}

func (j *JobInterfaceMock) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errors.New("JobInterfaceMock does not implement this")
}

func (j *JobInterfaceMock) Get(name string, options metav1.GetOptions) (*v1.Job, error) {
	dupeName := name
	j.GetCalledWith = &dupeName
	if j.GetError != nil {
		return nil, j.GetError
	} else {
		return j.GetResult, nil
	}
}

func (j *JobInterfaceMock) List(opts metav1.ListOptions) (*v1.JobList, error) {
	dupeOpts := opts
	j.ListCalledWith = &dupeOpts
	if j.ListError != nil {
		return nil, j.ListError
	} else {
		return j.ListResult, nil
	}

}

func (j *JobInterfaceMock) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("JobInterfaceMock does not implement this")
}

func (j *JobInterfaceMock) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Job, err error) {
	return nil, errors.New("JobInterfaceMock does not implement this")
}
