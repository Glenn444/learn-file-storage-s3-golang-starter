package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)


func processVideoForFastStart(inputfilePath string) (string, error)  {
	processedFilePath := fmt.Sprintf("%s.processing",inputfilePath)

   
	fmt.Printf("output file: %s\n",processedFilePath)
	cmd := exec.Command("ffmpeg","-i",inputfilePath,"-movflags","faststart","-codec","copy","-f","mp4",processedFilePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil{
		return  "",fmt.Errorf("failed to run ffmpeg: %s, %v",stderr.String(),err)
	}
	
	fileInfo,err := os.Stat(processedFilePath)
	if err != nil{
		return "",fmt.Errorf("could not stat processed file: %v",err)
	}

	if fileInfo.Size() == 0{
		return "",fmt.Errorf("processed file is empty")
	}

	return processedFilePath,nil
}