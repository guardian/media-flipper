package jobrunner

import (
	"errors"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PodInterfaceMock struct {
	fakeLogString string
}

func (p *PodInterfaceMock) Create(*v1.Pod) (*v1.Pod, error) {
	return nil, errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) Update(*v1.Pod) (*v1.Pod, error) {
	return nil, errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) UpdateStatus(*v1.Pod) (*v1.Pod, error) {
	return nil, errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) Delete(name string, options *metav1.DeleteOptions) error {
	return errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errors.New("mock does not implement this")
}

func (p *PodInterfaceMock) Get(name string, options metav1.GetOptions) (*v1.Pod, error) {
	return nil, errors.New("mock does not implement this")
}

func (p *PodInterfaceMock) List(opts metav1.ListOptions) (*v1.PodList, error) {
	return nil, errors.New("mock does not implement this")
}

func (p *PodInterfaceMock) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Pod, err error) {
	return nil, errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) GetEphemeralContainers(podName string, options metav1.GetOptions) (*v1.EphemeralContainers, error) {
	return nil, errors.New("mock does not implement this")
}
func (p *PodInterfaceMock) UpdateEphemeralContainers(podName string, ephemeralContainers *v1.EphemeralContainers) (*v1.EphemeralContainers, error) {
	return nil, errors.New("mock does not implement this")
}

func (p *PodInterfaceMock) Bind(binding *v1.Binding) error {
	return errors.New("mock does not implement this")
}

func (p *PodInterfaceMock) Evict(eviction *policy.Eviction) error {
	return errors.New("mock does not implement this")
}

func (p *PodInterfaceMock) GetLogs(name string, opts *v1.PodLogOptions) *restclient.Request {
	baseUrl, _ := url.Parse("https://fake-uri")
	cli := http.Client{}
	backoffMgr := restclient.NoBackoff{}
	rateLimiter := flowcontrol.NewFakeAlwaysRateLimiter()
	rq := restclient.NewRequest(&cli, "GET", baseUrl, "", restclient.ContentConfig{}, restclient.Serializers{}, &backoffMgr, rateLimiter, 1*time.Second)
	bodyContent := strings.NewReader(p.fakeLogString)
	rq.Body(bodyContent)
	return rq
}

//removed at the moment as i can't work out how to mock the extractLogs response
//func TestExtractLogs(t *testing.T){
//	podClient := PodInterfaceMock{}
//	podInfo := v1.Pod{}
//	_, err := extractLogs(&podInfo,&podClient)
//
//	if err != nil {
//		t.Errorf("extractLogs failed unexpectedly: %s", err)
//	} else {
//
//	}
//}
