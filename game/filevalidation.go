package game

import (
	"fmt"
	"log"
	"os"
)

func ValidateFilesExist() {
	folderList := []string{COMMON_FOLDER, COMMON_BIN_FOLDER, COMMON_DOOR_FOLDER, PL_FOLDER}
	for _, folderName := range folderList {
		folderExists, err := PathExists(folderName)
		if !folderExists {
			log.Fatal(fmt.Sprintf("Missing data error: Unable to find folder: %v. Error: ", folderName), err)
		}
	}

	regionSpecificFolders := map[string]string{
		"RDT_FOLDER":         RDT_FOLDER,
		"COMMON_DATA_FOLDER": COMMON_DATA_FOLDER,
	}
	for folderKey := range regionSpecificFolders {
		folderName := regionSpecificFolders[folderKey]
		folderExists, err := PathExists(folderName)
		if !folderExists {
			log.Fatal(
				fmt.Sprintf(
					"Missing data error: Unable to find region specific folder. "+
						"You might need to change the value of this variable of %v in resource.go. Current value of key %v = %v. Error: ",
					folderKey, folderKey, folderName,
				),
				err,
			)
		}
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
