package jobrunner

import (
	"errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/guardian/mediaflipper/common/models"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestFindRunnerFor(t *testing.T) {
	//FindRunnerFor should return a list of JobRunnerDesc for each item with a matching label
	mockJobClient := JobInterfaceMock{
		ListResult: &batchv1.JobList{
			Items: []batchv1.Job{
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       batchv1.JobSpec{},
					Status: batchv1.JobStatus{
						Active:    1,
						Succeeded: 0,
						Failed:    0,
					},
				},
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       batchv1.JobSpec{},
					Status: batchv1.JobStatus{
						Active:    0,
						Succeeded: 1,
						Failed:    0,
					},
				},
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       batchv1.JobSpec{},
					Status: batchv1.JobStatus{
						Active:    0,
						Succeeded: 0,
						Failed:    1,
					},
				},
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       batchv1.JobSpec{},
					Status: batchv1.JobStatus{
						Active:    0,
						Succeeded: 0,
						Failed:    0,
					},
				},
			},
		},
	}

	testId := uuid.MustParse("047EEAC1-F13F-4A13-865E-61974D48607E")
	result, err := FindRunnerFor(testId, &mockJobClient)

	if mockJobClient.ListCalledWith == nil {
		t.Error("List() was not called")
	} else {
		if mockJobClient.ListCalledWith.LabelSelector != "mediaflipper.jobStepId=047eeac1-f13f-4a13-865e-61974d48607e" {
			t.Error("List() was called with incorrect label selector, expected ")
		}
	}

	if err != nil {
		t.Error("FindRunnerFor failed unexpectedly: ", err)
	} else {
		if result == nil {
			t.Error("FindRunnerFor returned nil but no error!")
		} else {
			if len(*result) != 4 {
				t.Errorf("FindRunnerFor returned the wrong number of descriptors, expected 5 got %d", len(*result))
			}
			if (*result)[0].Status != models.CONTAINER_ACTIVE {
				t.Errorf("expected first item to be active (status %d) but got status %d", models.CONTAINER_ACTIVE, (*result)[0].Status)
			}
			if (*result)[1].Status != models.CONTAINER_COMPLETED {
				t.Errorf("expected second item to be completed (status %d) but got status %d", models.CONTAINER_COMPLETED, (*result)[0].Status)
			}
			if (*result)[2].Status != models.CONTAINER_FAILED {
				t.Errorf("expected third item to be failed (status %d) but got status %d", models.CONTAINER_FAILED, (*result)[0].Status)
			}
			if (*result)[3].Status != models.CONTAINER_FAILED {
				t.Errorf("expected fourth item to be failed (status %d) but got status %d", models.CONTAINER_FAILED, (*result)[0].Status)
			}
		}
	}
}

func TestFindRunnerFor_failing(t *testing.T) {
	//FindRunnerFor should pass on an error that occurs during the lookups
	mockJobClient := JobInterfaceMock{
		ListError: errors.New("kaboom!"),
	}

	testId := uuid.MustParse("047EEAC1-F13F-4A13-865E-61974D48607E")
	result, err := FindRunnerFor(testId, &mockJobClient)

	if mockJobClient.ListCalledWith == nil {
		t.Error("List() was not called")
	} else {
		if mockJobClient.ListCalledWith.LabelSelector != "mediaflipper.jobStepId=047eeac1-f13f-4a13-865e-61974d48607e" {
			t.Error("List() was called with incorrect label selector")
		}
	}

	if err == nil {
		t.Error("FindRunnerFor should have returned an error but got none")
	} else {
		if err.Error() != "kaboom!" {
			t.Errorf("FindRunnerFor returned wrong error, expected 'kaboom!' got %s", err.Error())
		}
	}
	if result != nil {
		t.Errorf("FindRunnerFor should have returned nil for result but got %s", spew.Sdump(result))
	}
}

func TestFindRunnerFor_empty(t *testing.T) {
	//FindRunnerFor should return a pointer to empty list if there are no results
	mockJobClient := JobInterfaceMock{
		ListResult: &batchv1.JobList{
			Items: []batchv1.Job{},
		},
	}

	testId := uuid.MustParse("047EEAC1-F13F-4A13-865E-61974D48607E")
	result, err := FindRunnerFor(testId, &mockJobClient)

	if mockJobClient.ListCalledWith == nil {
		t.Error("List() was not called")
	} else {
		if mockJobClient.ListCalledWith.LabelSelector != "mediaflipper.jobStepId=047eeac1-f13f-4a13-865e-61974d48607e" {
			t.Error("List() was called with incorrect label selector")
		}
	}
	if err != nil {
		t.Error("FindRunnerFor failed unexpectedly: ", err)
	} else {
		if result == nil {
			t.Error("FindRunnerFor returned nil but no error!")
		} else {
			if len(*result) != 0 {
				t.Errorf("Expected 0 results but got %d", len(*result))
			}
		}
	}
}

func TestFindServiceUrl(t *testing.T) {
	mockServiceClient := ServiceInterfaceMock{
		ListResponse: &corev1.ServiceList{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Items: []corev1.Service{
				corev1.Service{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "webapp-test",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							corev1.ServicePort{
								Name:     "other",
								Protocol: "tcp",
								Port:     1234,
							},
							corev1.ServicePort{
								Name:     "webapp",
								Protocol: "tcp",
								Port:     8888,
							},
							corev1.ServicePort{
								Name:     "something",
								Protocol: "tcp",
								Port:     4567,
							},
						},
					},
					Status: corev1.ServiceStatus{},
				},
			},
		},
	}

	result, err := FindServiceUrl(&mockServiceClient)
	if err != nil {
		t.Error("FindServiceUrl failed unexpectedly: ", err)
	}

	if result == nil {
		t.Error("FindServiceUrl returned nil without an error")
	} else {
		if *result != "http://webapp-test:8888" {
			t.Errorf("FindServiceUrl returned wrong url, expected http://webapp-test:8888 got %s", *result)
		}
	}
	if mockServiceClient.ListCalledWith == nil {
		t.Error("List() was not called")
	} else {
		if mockServiceClient.ListCalledWith.LabelSelector != "app=webapp,stack=MediaFlipper" {
			t.Errorf("List() was called with wrong label selector, expected app=webapp,stack=MediaFlipper got %s", mockServiceClient.ListCalledWith)
		}
	}
}

/**
FindServiceUrl should pass on any error that it encounters
*/
func TestFindServiceUrl_Error(t *testing.T) {
	mockServiceClient := ServiceInterfaceMock{
		ListError: errors.New("kaboom!"),
	}

	_, err := FindServiceUrl(&mockServiceClient)
	if err == nil {
		t.Error("FindServiceUrl did not return an error when it should")
	} else {
		if err.Error() != "kaboom!" {
			t.Errorf("FindServiceUrl returned wrong error, expected 'kaboom!' got %s", err.Error())
		}
	}
}

/**
FindServiceUrl should return an error if it finds no data
*/
func TestFindServiceUrl_Empty(t *testing.T) {
	mockServiceClient := ServiceInterfaceMock{
		ListResponse: &corev1.ServiceList{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Items:    []corev1.Service{},
		},
	}

	_, err := FindServiceUrl(&mockServiceClient)
	if err == nil {
		t.Error("FindServiceUrl did not return an error when it should")
	} else {
		if err.Error() != "Could not determine url, either no services match expected labels or no `webapp` port defined" {
			t.Errorf("FindServiceUrl returned wrong error, got %s", err.Error())
		}
	}
}
