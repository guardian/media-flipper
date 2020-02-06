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
	knownSettings map[uuid.UUID]JobSettings
}

/**
internal function to load the contents of a given yaml file
*/
func loadSettingsFromFile(fileName string) ([]JobSettings, error) {
	fileContent, readErr := ioutil.ReadFile(fileName)
	if readErr != nil {
		return nil, readErr
	}

	//try to unmarshal a list of settings
	var settingsList []JobSettings
	listMarshalErr := yaml.UnmarshalStrict(fileContent, &settingsList)
	if listMarshalErr == nil {
		return settingsList, nil
	}

	//if that didn't work, try to unmarshal a single setting
	var singleSetting JobSettings
	singleMarshalErr := yaml.UnmarshalStrict(fileContent, &singleSetting)
	if singleMarshalErr == nil {
		return []JobSettings{singleSetting}, nil
	}

	log.Printf("Could not unmarshal %s as settings list: %s", fileName, listMarshalErr)
	log.Printf("Could not unmarshal %s as single setting: %s", fileName, singleMarshalErr)
	return nil, errors.New("Could not read content, see logs for details.")
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

	mgr := TranscodeSettingsManager{knownSettings: make(map[uuid.UUID]JobSettings)}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			continue
		}
		moreSettings, readErr := loadSettingsFromFile(forPath + "/" + fileInfo.Name())
		if readErr == nil {
			log.Printf("Loaded %d settings from %s...", len(moreSettings), fileInfo.Name())
			for _, settingsData := range moreSettings {
				mgr.knownSettings[settingsData.SettingsId] = settingsData
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
func (mgr *TranscodeSettingsManager) GetSetting(forId uuid.UUID) *JobSettings {
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
func (mgr *TranscodeSettingsManager) ListSettings() *[]JobSettings {
	out := make([]JobSettings, len(mgr.knownSettings))
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
