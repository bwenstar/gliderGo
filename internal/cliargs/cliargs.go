// Package cliargs makes the flag package accept the argument order people type.
//
// Go's flag package stops at the first argument that is not a flag. That is documented and
// deliberate -- `go test ./... -run X` has to hand `-run X` to the test binary rather than
// to go's own parser -- but for a tool whose positional argument is a *file* the same rule is
// a trap, because the order a shell user types is the order they think in: name the thing,
// then say what to do with it. That is the order that did not work.
//
//	glidertool house stats "Demo House.house" -tier tutorial
//
// printed Demo House's eighteen rows with no tier column at all, and then stopped with
// `open -tier: no such file or directory`. Both halves of that are the expensive kind of
// wrong: the flag was not refused, it was silently demoted to a file name, and the diagnosis
// blames -tier for not existing on disk instead of for being in the wrong place.
// `house dump f.house -o out.txt` is worse, because its message is confidently false --
// `house dump takes exactly one house file`, said to somebody who gave exactly one house
// file. docs/IMPROVEMENTS.md 4.13 is the sweep that found this; the note there is that the
// ordering that fails is the one a shell user writes by habit.
//
// # Why this is a package and not a function in cmd/glidertool
//
// 4.13 proposed a `partitionArgs` in cmd/glidertool/main.go, and that would have been the
// smaller change. It is a package because both commands have the same argument shape --
// `glidergo Slumberland -scale 2` failed too, with `one house at a time: 3 were named
// (Slumberland, -scale, 2)` -- and because two copies of this would drift. The rule about
// what shares a package here is internal/project's: a nine-line formatter with two callers
// was left duplicated on purpose. This is not that. Deciding whether a flag eats the token
// after it means knowing about `-o=out`, about IsBoolFlag, about `--`, and about the bare `-`
// that means stdin, and a second copy that got one of those wrong would fail by moving an
// argument rather than by refusing one.
//
// # What it deliberately does not do
//
// It is not a parser, and every decision about what a flag *means* stays with the flag
// package. An unknown flag is left in the flag stream, so Parse still reports it by name.
// `-h` still reaches flag.ErrHelp. `-q true` still leaves `true` a positional, because bool
// flags take `-q=true` and nothing here changes that. A value is consumed verbatim whatever
// it looks like, so `-o --` takes `--` as the value, which is exactly what Parse does with
// it. The one question this package asks is the one flag's own parseOne asks -- does this
// flag want the next token -- and it asks the FlagSet rather than guessing, so a FlagSet and
// its reordering cannot disagree about a flag they both know.
package cliargs

import (
	"flag"
	"strings"
)

// FlagsFirst rewrites one command line so that Parse sees every flag before the first
// positional argument, and returns it. fs must already have its flags registered, because
// that is what is consulted; the args are not parsed and not validated here.
//
// The output is the flags in the order given, then `--`, then the positionals in the order
// given. The `--` is what keeps the move honest: without it a file that begins with a dash --
// which is only possible after the caller's own `--`, since that is the only way such a token
// reaches the positional list -- would be read back as a flag by the parser we are handing
// this to.
//
// Nothing is dropped and nothing is added except that one separator, so a command line that
// was already in the right order comes out meaning the same thing.
func FlagsFirst(fs *flag.FlagSet, args []string) []string {
	flags, files := make([]string, 0, len(args)), make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]

		// The caller's own terminator, honoured before anything else: after it the flag
		// package treats every token as positional however it is spelled, and a reordering
		// that looked past it would be inventing flags the caller had disclaimed.
		if a == "--" {
			files = append(files, args[i+1:]...)
			break
		}
		// "-" alone is not a flag, it is stdin -- `house build -` -- and len < 2 is what
		// parseOne itself tests, so this is the same rule rather than a matching one.
		if len(a) < 2 || a[0] != '-' {
			files = append(files, a)
			continue
		}

		flags = append(flags, a)
		if wantsValue(fs, a) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if len(files) == 0 {
		return flags
	}
	return append(append(flags, "--"), files...)
}

// wantsValue reports whether this flag token will take the token after it as its value.
//
// The three no answers are all no for different reasons, and none of them is a judgement
// about the flag: -o=out carries its own value, an unknown name has no value to want and is
// left for Parse to refuse by name, and a bool flag is the case the flag package itself
// decides with IsBoolFlag. Asking the FlagSet rather than keeping a list of this project's
// bool flags is the point -- a new -whatever added to a subcommand needs no edit here, and
// cannot be got wrong here either.
func wantsValue(fs *flag.FlagSet, arg string) bool {
	name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	if strings.Contains(name, "=") {
		return false
	}
	// Includes the bad-syntax spellings -- `---x` arrives here as "-x" and `-=x` as "=x",
	// neither of which is a registered name -- so those are passed through untouched and
	// Parse produces its own `bad flag syntax` for them. Diagnosing them here would mean
	// owning a second opinion about a syntax this package does not define.
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return !ok || !b.IsBoolFlag()
}
