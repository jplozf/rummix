package main

// ----------------------------------------------------------------------------
// IMPORTS
// ----------------------------------------------------------------------------
import (
	"bytes"
	"log"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

// ----------------------------------------------------------------------------
// TYPES
// ----------------------------------------------------------------------------
type SoundManager struct {
	context     *oto.Context
	tileSound   []byte
	beepSound   []byte
	winnerSound []byte
	okSound     []byte
	nogoodSound []byte
}

// ----------------------------------------------------------------------------
// VARS
// ----------------------------------------------------------------------------
var soundMgr *SoundManager

// ----------------------------------------------------------------------------
// InitAudio()
// ----------------------------------------------------------------------------
// Initialize the audio context and load sound files into memory
func InitAudio() {
	op := &oto.NewContextOptions{}
	op.SampleRate = 44100
	op.ChannelCount = 2
	op.Format = oto.FormatSignedInt16LE

	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		log.Println("Can't initialize audio context :", err)
		return
	}
	<-ready // Wait until the audio context is ready

	// Read sound files into memory
	soundBytes := resourceTileMp3.Content()
	beepBytes := resourceBeepMp3.Content()
	winnerBytes := resourceWinnerMp3.Content()
	okBytes := resourceOkMp3.Content()
	nogoodBytes := resourceNogoodMp3.Content()
	soundMgr = &SoundManager{
		context:     ctx,
		tileSound:   soundBytes,
		beepSound:   beepBytes,
		winnerSound: winnerBytes,
		okSound:     okBytes,
		nogoodSound: nogoodBytes,
	}
}

// ----------------------------------------------------------------------------
// PlayTileSound()
// ----------------------------------------------------------------------------
func PlayTileSound() {
	if soundMgr == nil || soundMgr.tileSound == nil {
		return
	}

	go func() {
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.tileSound))
		if err != nil {
			return
		}

		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		for player.IsPlaying() {
			// Keep the goroutine alive until the sound finishes playing
		}
		player.Close()
	}()
}

// ----------------------------------------------------------------------------
// PlayBeepSound()
// ----------------------------------------------------------------------------
func PlayBeepSound() {
	if soundMgr == nil || soundMgr.beepSound == nil {
		return
	}

	go func() {
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.beepSound))
		if err != nil {
			return
		}

		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		for player.IsPlaying() {
			// Keep the goroutine alive until the sound finishes playing
		}
		player.Close()
	}()
}

// ----------------------------------------------------------------------------
// PlayWinnerSound()
// ----------------------------------------------------------------------------
func PlayWinnerSound() {
	if soundMgr == nil || soundMgr.winnerSound == nil {
		return
	}

	go func() {
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.winnerSound))
		if err != nil {
			return
		}

		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		for player.IsPlaying() {
			// Keep the goroutine alive until the sound finishes playing
		}
		player.Close()
	}()
}

// ----------------------------------------------------------------------------
// PlayOkSound()
// ----------------------------------------------------------------------------
func PlayOkSound() {
	if soundMgr == nil || soundMgr.okSound == nil {
		return
	}

	go func() {
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.okSound))
		if err != nil {
			return
		}

		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		for player.IsPlaying() {
			// Keep the goroutine alive until the sound finishes playing
		}
		player.Close()
	}()
}

// ----------------------------------------------------------------------------
// PlayNogoodSound()
// ----------------------------------------------------------------------------
func PlayNogoodSound() {
	if soundMgr == nil || soundMgr.nogoodSound == nil {
		return
	}

	go func() {
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.nogoodSound))
		if err != nil {
			return
		}

		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		for player.IsPlaying() {
			// Keep the goroutine alive until the sound finishes playing
		}
		player.Close()
	}()
}
