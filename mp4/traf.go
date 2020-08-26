package mp4

import (
	"errors"
	"io"
)

// TrafBox - Track Fragment Box (traf)
//
// Contained in : Movie Fragment Box (moof)
//
type TrafBox struct {
	Tfhd  *TfhdBox
	Tfdt  *TfdtBox
	Trun  *TrunBox
	boxes []Box
}

// DecodeTraf - box-specific decode
func DecodeTraf(hdr *boxHeader, startPos uint64, r io.Reader) (Box, error) {
	children, err := DecodeContainerChildren(hdr, startPos, startPos+hdr.size, r)
	if err != nil {
		return nil, err
	}
	t := &TrafBox{}
	for _, b := range children {
		err := t.AddChild(b)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// AddChild - add child box
func (t *TrafBox) AddChild(b Box) error {
	switch b.Type() {
	case "tfhd":
		t.Tfhd = b.(*TfhdBox)
	case "tfdt":
		t.Tfdt = b.(*TfdtBox)
	case "trun":
		if t.Trun != nil {
			return errors.New("There is already one trun box. Multiple trun boxes not supported")
		}
		t.Trun = b.(*TrunBox)
	default:
	}
	t.boxes = append(t.boxes, b)
	return nil
}

// Type - return box type
func (t *TrafBox) Type() string {
	return "traf"
}

// Size - return calculated size
func (t *TrafBox) Size() uint64 {
	return containerSize(t.boxes)
}

// Children - list of children boxes
func (t *TrafBox) Children() []Box {
	return t.boxes
}

// Encode - write box to w
func (t *TrafBox) Encode(w io.Writer) error {
	return EncodeContainer(t, w)
}

// OptimizeTfhdTrun - optimize trun by default values in tfhd box
func (t *TrafBox) OptimizeTfhdTrun() error {
	tfhd := t.Tfhd
	trun := t.Trun
	if len(trun.samples) == 0 {
		return errors.New("No samples in trun")
	}
	if len(trun.samples) == 1 {
		return nil // No need to optimize
	}
	allZeroCTO := true
	hasCommonDur := true
	commonDur := trun.samples[0].Dur
	firstSampleFlags := trun.samples[0].Flags
	hasCommonFlags := true
	commonSampleFlags := trun.samples[1].Flags
	hasCommonSize := true
	commonSize := trun.samples[0].Size
	for i, s := range trun.samples {
		if s.Cto != 0 {
			allZeroCTO = false
		}
		if s.Dur != commonDur {
			hasCommonDur = false
		}
		if i >= 1 {
			if s.Flags != commonSampleFlags {
				hasCommonFlags = false
			}
		}
		if s.Size != commonSize {
			hasCommonSize = false
		}
	}
	if allZeroCTO {
		trun.flags = trun.flags & ^sampleCTOPresentFlag
	}
	if hasCommonDur {
		// Set defaultSampleDuration in tfhd and remove from trun
		tfhd.Flags = tfhd.Flags | defaultSampleDurationPresent
		tfhd.DefaultSampleDuration = commonDur
		trun.flags = trun.flags & ^sampleDurationPresentFlag
	}
	if hasCommonFlags {
		if firstSampleFlags != commonSampleFlags {
			trun.firstSampleFlags = firstSampleFlags
			trun.flags |= firstSamplePresentFlag
		}
		tfhd.Flags = tfhd.Flags | defaultSampleFlagsPresent
		tfhd.DefaultSampleFlags = commonSampleFlags
		trun.flags = trun.flags & ^sampleFlagsPresentFlag
	}
	if hasCommonSize {
		// Set defaultSampleSize in tfhd and remove from trun
		tfhd.Flags = tfhd.Flags | defaultSampleSizePresent
		tfhd.DefaultSampleSize = commonSize
		trun.flags = trun.flags & ^sampleSizePresentFlag
	}
	return nil
}
