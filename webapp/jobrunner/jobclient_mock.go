package jobrunner

import (
	"errors"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type JobClientMock struct {
	ErrorResponse error
	JobsCreated   []*v1.Job
}

func (j *JobClientMock) Create(newJob *v1.Job) (*v1.Job, error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	j.JobsCreated = append(j.JobsCreated, newJob)
	return newJob, nil
}

func (j JobClientMock) Update(*v1.Job) (*v1.Job, error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	return nil, errors.New("Not implemented by mock)")
}

func (j JobClientMock) UpdateStatus(*v1.Job) (*v1.Job, error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	return nil, errors.New("Not implemented by mock)")
}

func (j JobClientMock) Delete(name string, options *metav1.DeleteOptions) error {
	if j.ErrorResponse != nil {
		return j.ErrorResponse
	}
	return errors.New("Not implemented by mock)")
}

func (j JobClientMock) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	if j.ErrorResponse != nil {
		return j.ErrorResponse
	}
	return errors.New("Not implemented by mock)")
}

func (j JobClientMock) Get(name string, options metav1.GetOptions) (*v1.Job, error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	return nil, errors.New("Not implemented by mock")
}
func (j JobClientMock) List(opts metav1.ListOptions) (*v1.JobList, error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	return nil, errors.New("Not implemented by mock)")
}
func (j JobClientMock) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	return nil, errors.New("Not implemented by mock)")
}
func (j JobClientMock) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Job, err error) {
	if j.ErrorResponse != nil {
		return nil, j.ErrorResponse
	}
	return nil, errors.New("Not implemented by mock)")
}
