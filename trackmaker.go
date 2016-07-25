package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fohristiwhirl/wavmaker"
)

var midi_freq [109]float64 = [109]float64{
   8.175799,    8.661957,    9.177024,    9.722718,   10.300861,   10.913382,   11.562326,   12.249857,
  12.978272,   13.750000,   14.567618,   15.433853,   16.351598,   17.323914,   18.354048,   19.445436,
  20.601722,   21.826764,   23.124651,   24.499715,   25.956544,   27.500000,   29.135235,   30.867706,
  32.703196,   34.647829,   36.708096,   38.890873,   41.203445,   43.653529,   46.249303,   48.999429,
  51.913087,   55.000000,   58.270470,   61.735413,   65.406391,   69.295658,   73.416192,   77.781746,
  82.406889,   87.307058,   92.498606,   97.998859,  103.826174,  110.000000,  116.540940,  123.470825,
 130.812783,  138.591315,  146.832384,  155.563492,  164.813778,  174.614116,  184.997211,  195.997718,
 207.652349,  220.000000,  233.081881,  246.941651,  261.625565,  277.182631,  293.664768,  311.126984,
 329.627557,  349.228231,  369.994423,  391.995436,  415.304698,  440.000000,  466.163762,  493.883301,
 523.251131,  554.365262,  587.329536,  622.253967,  659.255114,  698.456463,  739.988845,  783.990872,
 830.609395,  880.000000,  932.327523,  987.766603, 1046.502261, 1108.730524, 1174.659072, 1244.507935,
1318.510228, 1396.912926, 1479.977691, 1567.981744, 1661.218790, 1760.000000, 1864.655046, 1975.533205,
2093.004522, 2217.461048, 2349.318143, 2489.015870, 2637.020455, 2793.825851, 2959.955382, 3135.963488,
3322.437581, 3520.000000, 3729.310092, 3951.066410, 4186.009045,
}

type Instrument struct {
	notes [109]*wavmaker.WAV
	flags [109]bool
}


func name_to_midi(name string) int {		// Note that, if this returns 0, it means "no note" ; actual MIDI values of 0 are never returned

	// Accepts notes in the following formats: C4  C4#  C#4  C4b  Cb4

	var result, number, accidental int
	var letter string

	if len(name) == 2 {
		letter = string(name[0])
		letter = strings.ToUpper(letter)
		number = int(name[1]) - 48						// -48 is conversion of ASCII to int
	} else if len(name) == 3 {
		letter = string(name[0])
		letter = strings.ToUpper(letter)
		if name[1] == '#' || name[1] == 'b' {
			number = int(name[2]) - 48
			if name[1] == '#' {
				accidental = 1
			} else {
				accidental = -1
			}
		} else if name[2] == '#' || name[2] == 'b' {
			number = int(name[1]) - 48
			if name[2] == '#' {
				accidental = 1
			} else {
				accidental = -1
			}
		} else {
			return 0									// How we deal with errors in this function...
		}
	} else {
		return 0
	}

	// First we set the result as if we asked for C in the relevant octave...

	switch number {
		case 0: result = 12		// C0
		case 1: result = 24		// C1
		case 2:	result = 36		// C2
		case 3:	result = 48		// C3
		case 4:	result = 60		// C4
		case 5:	result = 72		// C5
		case 6:	result = 84		// C6
		case 7:	result = 96		// C7
		case 8:	result = 108	// C8
		default: return 0
	}

	// Now we adjust it for the actual note that was requested...

	switch letter {
		case "C": result += 0
		case "D": result += 2
		case "E": result += 4
		case "F": result += 5
		case "G": result += 7
		case "A": result += 9
		case "B": result += 11
		default: return 0
	}

	// Now take into account flat or sharp symbols...

	result += accidental

	if result < 0 || result >= 109 {
		return 0
	}

	return result
}



func (instrument *Instrument) addfile(notestring string, filename string) {

	note := name_to_midi(notestring)

	wav, err := wavmaker.Load(filename)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't load instrument file '%s'\n", filename)
		os.Exit(1)
	}

	instrument.notes[note] = wav
	instrument.flags[note] = true
}


func (i *Instrument) insert(wav *wavmaker.WAV, t_loc uint32, s_loc uint32, notestring string) {

	note := name_to_midi(notestring)	// A number between 0 and 108 (MIDI value corresponding to note)

	if i.notes[note] == nil {

		a := int(note)
		b := int(note)

		note_to_stretch := 0

		for {			// Find reference note (one with its flag set) which was loaded from a file
			a--
			b++

			if a >= 0 {
				if i.flags[a] {
					note_to_stretch = a
					break
				}
			}

			if b <= 108 {
				if i.flags[b] {
					note_to_stretch = b
					break
				}
			}

			if a <= 0 && b >= 108 {
				fmt.Fprintf(os.Stderr, "insert() couldn't find a reference note")
				return
			}
		}

		ins_freq := midi_freq[note]
		ref_freq := midi_freq[note_to_stretch]

		i.notes[note] = i.notes[note_to_stretch].StretchedRelative(ref_freq / ins_freq)
	}

	wav.Add(t_loc, i.notes[note], s_loc, i.notes[note].FrameCount())
}


func main() {
	var piano Instrument
	piano.addfile("G4", "piano.ff.G4.wav")

	output := wavmaker.New(44100 * 3)

	for i, s := range []string{"C4", "D4", "E4", "F4", "G4", "A4", "B4", "C5"} {
		pos := uint32(i) * 11025
		piano.insert(output, pos, 0, s)
	}

	output.Fade(0.1)

	output.Save("realtest.wav")
}
