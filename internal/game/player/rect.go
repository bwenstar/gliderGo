package player

// The handful of QuickDraw rectangle helpers the player code uses, from
// GliderPRO/Sources/RectUtils.c. They are here rather than in a shared package
// because Rect is defined here and because these four are all the player needs.

// Tall is RectTall: the height. QuickDraw rects are half-open vertically for this
// purpose even though a drawn line includes both endpoints.
func (r Rect) Tall() int16 { return r.Bottom - r.Top }

// Wide is RectWide: the width.
func (r Rect) Wide() int16 { return r.Right - r.Left }

// Offset is QOffsetRect: translate by (h, v). Note the argument order is
// horizontal first, unlike Point, whose vertical component comes first.
func (r Rect) Offset(h, v int16) Rect {
	return Rect{Top: r.Top + v, Left: r.Left + h, Bottom: r.Bottom + v, Right: r.Right + h}
}

// ZeroCorner is ZeroRectCorner: move the rect so its top-left is the origin,
// keeping its size. Used to turn an atlas rect into a position-independent size.
func (r Rect) ZeroCorner() Rect {
	return Rect{Top: 0, Left: 0, Bottom: r.Tall(), Right: r.Wide()}
}

// Link is the resolved destination of a transit object: where a mail slot, duct or
// transporter leads.
//
// In the original this is three globals filled in by an identical eleven-line block
// at the head of StartGliderMailingIn, StartGliderDuctingDown, StartGliderDuctingUp
// and StartGliderTransporting (Modes.c:163-171, :226-234, :259-267, :301-309). The
// block walks masterObjects to get a room and object index, asks
// WhatAreWeLinkedTo, and reads the destination object's rect. All of that is object
// graph work, so in the port the caller resolves it and passes the answer in --
// which also means these functions no longer need the house at all.
type Link struct {
	Room int16 // transRoom: the destination room's index
	Rect Rect  // transRect: the destination object's rect
	What int16 // linkedToWhat: one of the LinkedTo* constants
}

// What a transit object is linked to (GliderDefines.h:610-614). Only
// LinkedToLeftMailbox is compared against by the player code; the rest are here
// because they share the enumeration and 1.5 will need them.
const (
	LinkedToOther        int16 = 0
	LinkedToLeftMailbox  int16 = 1
	LinkedToRightMailbox int16 = 2
	LinkedToCeilingDuct  int16 = 3
	LinkedToFloorDuct    int16 = 4
)
