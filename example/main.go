package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/GanymedeNil/go-webrtcvad"
	"github.com/cryptix/wav"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("usage: example infile.wav")
	}

	filename := flag.Arg(0)

	info, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	wavReader, err := wav.NewReader(file, info.Size())
	if err != nil {
		log.Fatal(err)
	}
	reader, err := wavReader.GetDumbReader()
	if err != nil {
		log.Fatal(err)
	}

	wavInfo := wavReader.GetFile()
	rate := int(wavInfo.SampleRate)
	if wavInfo.Channels != 1 {
		log.Fatal("expected mono file")
	}
	if rate != 16000 {
		log.Fatal("expected 16kHz file")
	}

	vad, err := webrtcvad.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := vad.SetMode(2); err != nil {
		log.Fatal(err)
	}

	frame := make([]byte, 320)
	if ok := vad.ValidRateAndFrameLength(rate, len(frame)); !ok {
		log.Fatal("invalid rate or frame length")
	}

	var isActive bool
	var offset int
	var duration time.Duration
	var splitTime time.Duration
	var st time.Duration = 0
	var tmpbuffer []byte
	report := func() {
		t := time.Duration(offset) * time.Second / time.Duration(rate) / 2
		splitTime = t
		//fmt.Printf("isActive = %v, t = %v\n", isActive, t)
	}

	for {
		_, err := io.ReadFull(reader, frame)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		frameActive, err := vad.Process(rate, frame)
		if err != nil {
			log.Fatal(err)
		}

		tmpbuffer = append(tmpbuffer, frame...)

		if isActive != frameActive || offset == 0 {
			isActive = frameActive
			var tmpTime = splitTime
			if st==0 {
				st = splitTime
			}
			report()
			if !frameActive {
				duration += splitTime - tmpTime
				if duration>(2*time.Second) {
					fmt.Printf("st--%s,et--%s,du--%s\n",st,splitTime,time.Duration(len(tmpbuffer))*time.Second/time.Duration(rate)/2)
					writeWave(tmpbuffer, st, splitTime)
					fmt.Println("len--", len(tmpbuffer))
					tmpbuffer = []byte{}
					duration = 0
					st = 0
					fmt.Println("clear")
				}


			}
		}


		offset += len(frame)
	}

	report()
}

func writeWave(buffer []byte, start, end time.Duration) {
	filename := fmt.Sprintf("tmp/chunk-%v-%v.wav", start.Nanoseconds()/1e6, end.Nanoseconds()/1e6)
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		log.Println("create file err:", err)
	}
	meta := wav.File{
		Channels:        1,
		SampleRate:      16000,
		SignificantBits: 16,
	}

	writer, err := meta.NewWriter(f)
	_, err = writer.Write(buffer)
	if err != nil {
		log.Println(err)
	}
	err = writer.Close()
	if err != nil {
		log.Println(err)
	}
}
