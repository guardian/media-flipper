package helpers

import (
	"errors"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	v1batch "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/rest"
)

type K8ClientWrapper interface {
	BatchV1() v1batch.BatchV1Interface
}

type K8JobsMock struct {
}

func (j K8JobsMock) RESTClient() rest.Interface {
	return nil
}

func (j K8JobsMock) Jobs(namespace string) v1batch.JobInterface {
	return NewK8JobsMock()
}

type K8JobOpsMock struct {
	callCount       map[string]int
	FailWith        error
	DeletedNameList []string
	KnownJobs       map[string]*v1.Job
}

func NewK8JobsMock() K8JobOpsMock {
	return K8JobOpsMock{callCount: map[string]int{
		"create":           0,
		"update":           0,
		"updatestatus":     0,
		"delete":           0,
		"deletecollection": 0,
		"get":              0,
		"list":             0,
		"watch":            0,
		"patch":            0,
	}}

}

func (j K8JobOpsMock) Create(newJob *v1.Job) (*v1.Job, error) {
	j.callCount["create"] += 1
	if j.FailWith != nil {
		return nil, j.FailWith
	} else {
		return newJob, nil
	}
}
func (j K8JobOpsMock) Update(newJob *v1.Job) (*v1.Job, error) {
	j.callCount["create"] += 1
	if j.FailWith != nil {
		return nil, j.FailWith
	} else {
		return newJob, nil
	}
}
func (j K8JobOpsMock) UpdateStatus(newJob *v1.Job) (*v1.Job, error) {
	j.callCount["create"] += 1
	if j.FailWith != nil {
		return nil, j.FailWith
	} else {
		return newJob, nil
	}
}
func (j K8JobOpsMock) Delete(name string, options *metav1.DeleteOptions) error {
	j.DeletedNameList = append(j.DeletedNameList, name)
	j.callCount["delete"] += 1
	if j.FailWith != nil {
		return j.FailWith
	} else {
		return nil
	}
}
func (j K8JobOpsMock) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	j.callCount["deletecollection"] += 1
	if j.FailWith != nil {
		return j.FailWith
	} else {
		return nil
	}
}
func (j K8JobOpsMock) Get(name string, options metav1.GetOptions) (*v1.Job, error) {
	j.callCount["get"] += 1
	if j.FailWith != nil {
		return nil, j.FailWith
	} else {
		return j.KnownJobs[name], nil
	}
}
func (j K8JobOpsMock) List(opts metav1.ListOptions) (*v1.JobList, error) {
	jobList := make([]v1.Job, len(j.KnownJobs))
	i := 0
	for _, jb := range j.KnownJobs {
		jobList[i] = *jb
		i += 1
	}

	rtn := v1.JobList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items:    jobList,
	}
	j.callCount["list"] += 1
	if j.FailWith != nil {
		return nil, j.FailWith
	} else {
		return &rtn, nil
	}
}
func (j K8JobOpsMock) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("mock does not implement watch()")
}
func (j K8JobOpsMock) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Job, err error) {
	return nil, errors.New("mock does not implement patch()")
}
