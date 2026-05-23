package main

import (
	"bytes"
	"log"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

type SoundManager struct {
	context     *oto.Context
	tileSound   []byte
	beepSound   []byte
	winnerSound []byte
	okSound     []byte
	nogoodSound []byte
}

var soundMgr *SoundManager

// Initialise le moteur audio (à appeler une seule fois au démarrage dans ta fonction main)
func InitAudio() {
	op := &oto.NewContextOptions{}
	op.SampleRate = 44100
	op.ChannelCount = 2
	op.Format = oto.FormatSignedInt16LE
	// op.Format = oto.FormatFloat32LE // Standard pour Oto v3

	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		log.Println("Impossible d'initialiser le moteur audio:", err)
		return
	}
	<-ready // Attend que le périphérique audio soit prêt

	// Lire le fichier de son (ex: clic de tuile)
	soundBytes := resourceTileMp3.Content()    // Assure-toi que resourceTileMp3 est défini dans ton code
	beepBytes := resourceBeepMp3.Content()     // Assure-toi que resourceBeepMp3 est défini dans ton code
	winnerBytes := resourceWinnerMp3.Content() // Assure-toi que resourceWinnerMp3 est défini dans ton code
	okBytes := resourceOkMp3.Content()         // Assure-toi que resourceOkMp3 est défini dans ton code
	nogoodBytes := resourceNogoodMp3.Content() // Assure-toi que resourceNogoodMp3 est défini dans ton code
	soundMgr = &SoundManager{
		context:     ctx,
		tileSound:   soundBytes,
		beepSound:   beepBytes,
		winnerSound: winnerBytes,
		okSound:     okBytes,
		nogoodSound: nogoodBytes,
	}
}

// Joue le son du clic de manière asynchrone (sans bloquer le jeu)
func PlayTileSound() {
	if soundMgr == nil || soundMgr.tileSound == nil {
		return
	}

	go func() {
		// Décode le MP3 à la volée depuis la mémoire
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.tileSound))
		if err != nil {
			return
		}

		// Crée le joueur et lance la lecture
		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		// Attend la fin du son pour fermer le player et libérer la mémoire
		for player.IsPlaying() {
			// On laisse le son se jouer
		}
		player.Close()
	}()
}

func PlayBeepSound() {
	if soundMgr == nil || soundMgr.beepSound == nil {
		return
	}

	go func() {
		// Décode le MP3 à la volée depuis la mémoire
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.beepSound))
		if err != nil {
			return
		}

		// Crée le joueur et lance la lecture
		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		// Attend la fin du son pour fermer le player et libérer la mémoire
		for player.IsPlaying() {
			// On laisse le son se jouer
		}
		player.Close()
	}()
}

func PlayWinnerSound() {
	if soundMgr == nil || soundMgr.winnerSound == nil {
		return
	}

	go func() {
		// Décode le MP3 à la volée depuis la mémoire
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.winnerSound))
		if err != nil {
			return
		}

		// Crée le joueur et lance la lecture
		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		// Attend la fin du son pour fermer le player et libérer la mémoire
		for player.IsPlaying() {
			// On laisse le son se jouer
		}
		player.Close()
	}()
}

func PlayOkSound() {
	if soundMgr == nil || soundMgr.okSound == nil {
		return
	}

	go func() {
		// Décode le MP3 à la volée depuis la mémoire
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.okSound))
		if err != nil {
			return
		}

		// Crée le joueur et lance la lecture
		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		// Attend la fin du son pour fermer le player et libérer la mémoire
		for player.IsPlaying() {
			// On laisse le son se jouer
		}
		player.Close()
	}()
}

func PlayNogoodSound() {
	if soundMgr == nil || soundMgr.nogoodSound == nil {
		return
	}

	go func() {
		// Décode le MP3 à la volée depuis la mémoire
		decoded, err := mp3.NewDecoder(bytes.NewReader(soundMgr.nogoodSound))
		if err != nil {
			return
		}

		// Crée le joueur et lance la lecture
		player := soundMgr.context.NewPlayer(decoded)
		player.Play()

		// Attend la fin du son pour fermer le player et libérer la mémoire
		for player.IsPlaying() {
			// On laisse le son se jouer
		}
		player.Close()
	}()
}
