package models

import (
	"errors"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type TranscodeSettingsManager struct {
	knownSettings map[uuid.UUID]TranscodeTypeSettings
}

func attemptUnmarshalJobSettings(from map[string]interface{}) (TranscodeTypeSettings, error) {
	var result JobSettings
	marshalErr := CustomisedMapStructureDecode(from, &result)
	return result, marshalErr
}

func attemptUnmarshalImageSettings(from map[string]interface{}) (TranscodeTypeSettings, error) {
	var result TranscodeImageSettings
	marshalErr := CustomisedMapStructureDecode(from, &result)
	return result, marshalErr
}

/**
internal function to load the contents of a given yaml file
*/
func loadSettingsFromFile(fileName string) ([]TranscodeTypeSettings, error) {
	fileContent, readErr := ioutil.ReadFile(fileName)
	if readErr != nil {
		return nil, readErr
	}

	//try to unmarshal a list of settings
	var settingsList []map[string]interface{}

	extractErr := yaml.Unmarshal(fileContent, &settingsList)
	if extractErr != nil {
		return nil, extractErr
	}

	rtn := make([]TranscodeTypeSettings, len(settingsList))

	for i, rawSetting := range settingsList {
		jobSetting, jsErr := attemptUnmarshalJobSettings(rawSetting)
		if jsErr == nil {
			//log.Printf("got transcode settings: %s", spew.Sdump(jobSetting))
			rtn[i] = jobSetting
			continue
		} else {
			//log.Printf("could not read in as job: %s", jsErr)
		}

		imageSetting, isErr := attemptUnmarshalImageSettings(rawSetting)
		if isErr == nil {
			//log.Printf("got image settings: %s", spew.Sdump(imageSetting))
			rtn[i] = imageSetting
			continue
		} else {
			//log.Printf("could not read in as image: %s", isErr)
		}
	}
	return nil, errors.New("could not read content, see logs for details")
}

/**
initialise the TranscodeSettingsManager by reading all yaml settings files in the provided directory
*/
func NewTranscodeSettingsManager(forPath string) (*TranscodeSettingsManager, error) {
	pathStatInfo, statErr := os.Stat(forPath)
	if statErr != nil {
		return nil, statErr
	}
	if !pathStatInfo.IsDir() {
		return nil, errors.New("TranscodeSettingsManager requires a directory")
	}

	files, err := ioutil.ReadDir(forPath)
	if err != nil {
		return nil, err
	}

	mgr := TranscodeSettingsManager{knownSettings: make(map[uuid.UUID]TranscodeTypeSettings)}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue
		}
		moreSettings, readErr := loadSettingsFromFile(forPath + "/" + fileInfo.Name())
		if readErr == nil {
			log.Printf("Loaded %d settings from %s...", len(moreSettings), fileInfo.Name())
			for _, settingsData := range moreSettings {
				mgr.knownSettings[settingsData.GetId()] = settingsData
			}
		} else {
			log.Printf("Could not read in settings from %s", forPath+"/"+fileInfo.Name())
		}
	}
	return &mgr, nil
}

/**
returns a setting for the given ID, or nil if it is not found
*/
func (mgr *TranscodeSettingsManager) GetSetting(forId uuid.UUID) *TranscodeTypeSettings {
	result, gotIt := mgr.knownSettings[forId]
	if gotIt {
		return &result
	} else {
		return nil
	}
}

/**
returns a list of all the known settings
*/
func (mgr *TranscodeSettingsManager) ListSettings() *[]TranscodeTypeSettings {
	out := make([]TranscodeTypeSettings, len(mgr.knownSettings))
	i := 0
	for _, s := range mgr.knownSettings {
		out[i] = s
		i += 1
	}
	return &out
}

func (mgr *TranscodeSettingsManager) ListSummary() *[]JobSettingsSummary {
	out := make([]JobSettingsSummary, len(mgr.knownSettings))
	i := 0
	for _, s := range mgr.knownSettings {
		out[i] = s.Summarise()
	}
	return &out
}
