package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
)

type FFProbeOutput struct{
	Streams []Stream `json:"streams"`
}

type Stream struct{
	Width int `json:"width"`
	Height int `json:"height"`
}

func getVideoAspectRatio(filePath string) (string,error)  {
	cmd := exec.Command("ffprobe","-v","error","-print_format","json","-show_streams",filePath)
	
	var b bytes.Buffer
	cmd.Stdout = &b

	err := cmd.Run()
	if err != nil{
		log.Fatal(err)
		return  "",fmt.Errorf("failed to run ffprobe: %w",err)
	}

	
	var videoAspect FFProbeOutput
	

	err = json.Unmarshal(b.Bytes(),&videoAspect)
	if err != nil{
		return "",fmt.Errorf("failed to parse ffprobe output")
	}
	
	for _,stream := range videoAspect.Streams{
		if stream.Width > 0 && stream.Height > 0{
			ratio := float64(stream.Width) / float64(stream.Height)

			//Determine which standard aspect ratio it matches

			if math.Abs(ratio-16.0/9.0) < 0.1{
				return  "16:9",nil
			}else if math.Abs(ratio-9.0/16.0) < 0.1{
				return  "9:16",nil
			}else{
				return "other",nil
			}
		}
	}

	return  "",fmt.Errorf("no video stream with dimensions found")
}