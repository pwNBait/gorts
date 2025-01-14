# Tcl on Windows has unfortunate defaults:
#   - cp1252 encoding, which will mangle utf-8 source code
#   - crlf linebreaks instead of unix-style lf
# Let's be consistent cross-platform to avoid surprises:
encoding system "utf-8"
foreach p {stdin stdout stderr} {
    fconfigure $p -encoding "utf-8"
    fconfigure $p -translation lf
}

package require Tk

wm title . "Overly Repetitive Tedious Software (in Go)"
tk appname gorts

# Proper Windows theme doesn't allow setting fieldbackground on text inputs,
# so let's settle with `clam` instead.
ttk::style theme use clam

wm protocol . WM_DELETE_WINDOW {
    exit 0
}

# Data that we send to the actual web-based overlay:
array set scoreboard {
    description ""
    subtitle ""
    stage ""
    p1name ""
    p1country ""
    p1score 0
    p1team ""
    p1character ""
    p2name ""
    p2country ""
    p2score 0
    p2team ""
    p2character ""
    c1title ""
    c1subtitle ""
    c2title ""
    c2subtitle ""
    p1team ""
}

# $applied_scoreboard represents data that has actually been applied
# to the overlay. This is used to display diff in the UI, and to restore data
# when user clicks "Discard".
foreach key [array names scoreboard] {
    set applied_scoreboard($key) scoreboard($key)
}

array set var_to_widget {
    description .n.m.description.entry
    subtitle .n.m.subtitle.entry
    stage .n.m.stage.entry
    p1name .n.m.players.p1name
    p1country .n.m.players.p1country
    p1score .n.m.players.p1score
    p1team .n.m.players.p1team
    p1character .n.m.players.p1character
    p2name .n.m.players.p2name
    p2country .n.m.players.p2country
    p2score .n.m.players.p2score
    p2team .n.m.players.p2team
    p2character .n.m.players.p2character
    c1title .n.l.c1title.entry
    c1subtitle .n.l.c1subtitle.entry
    c2title .n.l.c2title.entry
    c2subtitle .n.l.c2subtitle.entry
}

array set startgg {
    token ""
    slug ""
    phasegroupid ""
    msg ""
}

# GUI has 2 tabs: Main (.n.m) and start.gg (.n.s)

ttk::notebook .n
ttk::frame .n.m -padding 5
ttk::frame .n.s -padding 5
ttk::frame .n.l -padding 5
.n add .n.m -text Main
.n add .n.s -text start.gg
.n add .n.l -text "Lower Thirds"
grid .n -column 0 -row 0 -sticky NESW

# Main tab:

ttk::frame .n.m.description
ttk::label .n.m.description.lbl -text "Title"
ttk::entry .n.m.description.entry -textvariable scoreboard(description)
ttk::frame .n.m.subtitle
ttk::label .n.m.subtitle.lbl -text "Subtitle"
ttk::entry .n.m.subtitle.entry -textvariable scoreboard(subtitle)
ttk::frame .n.m.players
ttk::label .n.m.players.p1lbl -text "Player 1"
ttk::combobox .n.m.players.p1name -textvariable scoreboard(p1name) -width 35
ttk::combobox .n.m.players.p1country -textvariable scoreboard(p1country) -width 5
ttk::spinbox .n.m.players.p1score -textvariable scoreboard(p1score) -from 0 -to 999 -width 4
ttk::button .n.m.players.p1win -text "▲ Win" -width 6 -command {incr scoreboard(p1score)}
ttk::label .n.m.players.p1teamlbl -text "Team 1"
ttk::combobox .n.m.players.p1team -textvariable scoreboard(p1team) -width 15
ttk::combobox .n.m.players.p1character -textvariable scoreboard(p1character) -width 15
ttk::separator .n.m.players.separator -orient horizontal
ttk::label .n.m.players.p2lbl -text "Player 2"
ttk::combobox .n.m.players.p2name -textvariable scoreboard(p2name) -width 35
ttk::combobox .n.m.players.p2country -textvariable scoreboard(p2country) -width 5
ttk::spinbox .n.m.players.p2score -textvariable scoreboard(p2score) -from 0 -to 999 -width 4
ttk::button .n.m.players.p2win -text "▲ Win" -width 6 -command {incr scoreboard(p2score)}
ttk::label .n.m.players.p2teamlbl -text "Team 2"
ttk::combobox .n.m.players.p2team -textvariable scoreboard(p2team) -width 15
ttk::combobox .n.m.players.p2character -textvariable scoreboard(p2character) -width 15
ttk::frame .n.m.stage
ttk::label .n.m.stage.lbl -text "Stage"
ttk::combobox .n.m.stage.entry -textvariable scoreboard(stage) -width 35
ttk::frame .n.m.buttons
ttk::button .n.m.buttons.apply -text "▶ Apply" -command applyscoreboard
ttk::button .n.m.buttons.discard -text "✖ Discard" -command discardscoreboard
ttk::button .n.m.buttons.reset -text "↶ Reset scores" -command {
    set scoreboard(p1score) 0
    set scoreboard(p2score) 0
}
ttk::button .n.m.buttons.swap -text "⇄ Swap players" -command {
    # Since country is updated whenever name is updated, we'll need to write
    # countries last.
    set p1country $scoreboard(p1country)
    set p2country $scoreboard(p2country)
    foreach key {name score team} {
        set tmp $scoreboard(p1$key)
        set scoreboard(p1$key) $scoreboard(p2$key)
        set scoreboard(p2$key) $tmp
    }
    set scoreboard(p1country) $p2country
    set scoreboard(p2country) $p1country
}
ttk::button .n.m.buttons.sggstreamqueue -text "Get Latest from StartGG" -command getstreamqueue
ttk::label .n.m.status -textvariable mainstatus
grid .n.m.description -row 0 -column 0 -sticky NESW -pady {0 5}
grid .n.m.description.lbl -row 0 -column 0 -padx {0 5}
grid .n.m.description.entry -row 0 -column 1 -sticky EW
grid columnconfigure .n.m.description 1 -weight 1
grid .n.m.subtitle -row 1 -column 0 -sticky NESW -pady {0 5}
grid .n.m.subtitle.lbl -row 0 -column 0 -padx {0 5}
grid .n.m.subtitle.entry -row 0 -column 1 -sticky EW
grid columnconfigure .n.m.subtitle 1 -weight 1
grid .n.m.players -row 2 -column 0
grid .n.m.players.p1lbl -row 0 -column 0
grid .n.m.players.p1name -row 0 -column 1
grid .n.m.players.p1country -row 0 -column 2
grid .n.m.players.p1score -row 0 -column 3
grid .n.m.players.p1win -row 0 -column 4 -padx {5 0} -rowspan 2 -sticky NS
grid .n.m.players.p1teamlbl -row 1 -column 0
grid .n.m.players.p1team -row 1 -column 1 -sticky EW
grid .n.m.players.p1character -row 1 -column 2 -columnspan 2 -sticky EW
grid .n.m.players.separator -row 2 -column 0 -columnspan 5 -pady 10 -sticky EW
grid .n.m.players.p2lbl -row 3 -column 0
grid .n.m.players.p2name -row 3 -column 1
grid .n.m.players.p2country -row 3 -column 2
grid .n.m.players.p2score -row 3 -column 3
grid .n.m.players.p2win -row 3 -column 4 -padx {5 0} -rowspan 2 -sticky NS
grid .n.m.players.p2teamlbl -row 4 -column 0
grid .n.m.players.p2team -row 4 -column 1 -sticky EW
grid .n.m.players.p2character -row 4 -column 2 -columnspan 2 -sticky EW
grid .n.m.stage -row 3 -column 0 -sticky NESW -pady {10 5}
grid .n.m.stage.lbl -row 0 -column 0 -padx {0 5}
grid .n.m.stage.entry -row 0 -column 1 -sticky NW
grid columnconfigure .n.m.stage 1 -weight 1
grid .n.m.buttons -row 4 -column 0 -sticky W -pady {10 0}
grid .n.m.buttons.apply -row 0 -column 0
grid .n.m.buttons.discard -row 0 -column 1
grid .n.m.buttons.reset -row 0 -column 2
grid .n.m.buttons.swap -row 0 -column 3
grid .n.m.buttons.sggstreamqueue -row 0 -column 4
grid .n.m.status -row 5 -column 0 -columnspan 5 -pady {10 0} -sticky EW
grid columnconfigure .n.m.players 2 -pad 5
grid columnconfigure .n.m.buttons 1 -pad 15
grid columnconfigure .n.m.buttons 3 -pad 15
grid rowconfigure .n.m.players 1 -pad 5
grid rowconfigure .n.m.players 3 -pad 5

# start.gg tab:

#.n select .n.s; # for debug only
ttk::label .n.s.tokenlbl -text "Personal token: "
ttk::entry .n.s.token -show * -textvariable startgg(token)
ttk::label .n.s.tournamentlbl -text "Tournament slug: "
ttk::entry .n.s.tournamentslug -textvariable startgg(slug)
ttk::label .n.s.phasegrouplbl -text "Phase group id: "
ttk::entry .n.s.phasegroupid -textvariable startgg(phasegroupid)
ttk::frame .n.s.buttons
ttk::button .n.s.buttons.fetch -text "↓ Fetch players" -command fetchplayers
ttk::button .n.s.buttons.bracket -text "↓ Fetch bracket" -command getbracket
ttk::button .n.s.buttons.clear -text "✘ Clear" -command clearstartgg
ttk::label .n.s.msg -textvariable startgg(msg)

grid .n.s.tokenlbl -row 0 -column 0 -sticky W
grid .n.s.token -row 0 -column 1 -sticky EW
grid .n.s.tournamentlbl -row 1 -column 0 -sticky W
grid .n.s.tournamentslug -row 1 -column 1 -sticky EW
grid .n.s.phasegrouplbl -row 2 -column 0 -sticky W
grid .n.s.phasegroupid -row 2 -column 1 -sticky EW
grid .n.s.buttons -row 3 -column 1 -stick WE
grid .n.s.buttons.fetch -stick W
grid .n.s.buttons.bracket -row 0 -column 1 -stick W -padx 5
grid .n.s.buttons.clear -row 0 -column 2 -stick W -padx 5
grid .n.s.msg -row 4 -column 1 -stick W
grid columnconfigure .n.s 1 -weight 1
grid rowconfigure .n.s 1 -pad 5
grid rowconfigure .n.s 2 -pad 5

# Lower Thirds tab:

ttk::frame .n.l.c1title
ttk::label .n.l.c1title.lbl -text "Commentary One"
ttk::entry .n.l.c1title.entry -textvariable scoreboard(c1title)
ttk::frame .n.l.c1subtitle
ttk::label .n.l.c1subtitle.lbl -text "Subtitle One"
ttk::entry .n.l.c1subtitle.entry -textvariable scoreboard(c1subtitle)
ttk::frame .n.l.c2title
ttk::label .n.l.c2title.lbl -text "Commentary Two"
ttk::entry .n.l.c2title.entry -textvariable scoreboard(c2title)
ttk::frame .n.l.c2subtitle
ttk::label .n.l.c2subtitle.lbl -text "Subtitle One"
ttk::entry .n.l.c2subtitle.entry -textvariable scoreboard(c2subtitle)

grid .n.l.c1title -row 0 -column 0 -sticky NESW -pady {0 5}
grid .n.l.c1title.lbl -row 0 -column 0 -padx {0 5}
grid .n.l.c1title.entry -row 0 -column 1 -sticky EW
grid columnconfigure .n.l.c1title 1 -weight 1
grid .n.l.c1subtitle -row 1 -column 0 -sticky NESW -pady {0 5}
grid .n.l.c1subtitle.lbl -row 1 -column 0 -padx {0 5}
grid .n.l.c1subtitle.entry -row 1 -column 1 -sticky EW
grid .n.l.c2title -row 0 -column 2 -sticky NESW -pady {0 5}
grid .n.l.c2title.lbl -row 0 -column 2 -padx {0 5}
grid .n.l.c2title.entry -row 0 -column 3 -sticky EW
grid columnconfigure .n.l.c2title 2 -weight 1
grid .n.l.c2subtitle -row 1 -column 2 -sticky NESW -pady {0 5}
grid .n.l.c2subtitle.lbl -row 1 -column 2 -padx {0 5}
grid .n.l.c2subtitle.entry -row 1 -column 3 -sticky EW
grid rowconfigure .n.l 1 -pad 5
grid rowconfigure .n.l 2 -pad 5

proc initialize {} {
    loadicon
    loadstartgg
    loadwebmsg
    loadcountrycodes
    loadscoreboard
    loadplayernames

    setupdiffcheck
    setupplayersuggestion
    setupcharacters
    setupstages


    # By default this window is not focused and not even brought to
    # foreground on Windows. I suspect it's because tcl is exec'ed from Go.
    # The old "iconify, deiconify" trick no longer seems to work, so this time
    # I'm passing it to Go to call the winapi's SetForegroundWindow directly.
    if {$::tcl_platform(platform) == "windows"} {
        windows_forcefocus
    }
}

# Very simple line-based IPC system where Tcl client talks to Go server
# via stdin/stdout
proc ipc_write {method args} {
    puts "$method [llength $args]"
    foreach a $args {
        puts "$a"
    }
}
proc ipc_read {} {
    set results {}
    set numlines [gets stdin]
    for {set i 0} {$i < $numlines} {incr i} {
        lappend results [gets stdin]
    }
    return $results
}
proc ipc {method args} {
    ipc_write $method {*}$args
    return [ipc_read]
}

proc windows_forcefocus {} {
    # First call winapi's SetForegroundWindow()
    set handle [winfo id .]
    ipc "forcefocus" $handle
    # Then call force focus on tcl side
    focus -force .
    # We must do both in order to properly focus on main tk window.
    # Don't ask me why - that's just how it works.
    #
    # Alternatively we can try making Tcl our entrypoint instead of exec-ing
    # Tcl from Go. Maybe some other time.
}

proc loadicon {} {
    set iconblob [image create photo -file gorts.png]
    wm iconphoto . -default $iconblob
}

proc loadstartgg {} {
    set resp [ipc "getstartgg"]
    set ::startgg(token) [lindex $resp 0]
    set ::startgg(slug) [lindex $resp 1]
    set ::startgg(phasegroupid) [lindex $resp 2]
}

proc loadwebmsg {} {
    set resp [ipc "getwebport"]
    set webport [lindex $resp 0]
    set ::mainstatus "Point your OBS browser source to http://localhost:${webport}"
}

proc loadcountrycodes {} {
    set codes [ipc "getcountrycodes"]
    .n.m.players.p1country configure -values $codes
    .n.m.players.p2country configure -values $codes
}

proc loadscoreboard {} {
    set sb [ipc "getscoreboard"]
    set ::scoreboard(description) [lindex $sb 0]
    set ::scoreboard(subtitle) [lindex $sb 1]
    set ::scoreboard(p1name) [lindex $sb 2]
    set ::scoreboard(p1country) [lindex $sb 3]
    set ::scoreboard(p1score) [lindex $sb 4]
    set ::scoreboard(p1team) [lindex $sb 5]
    set ::scoreboard(p2name) [lindex $sb 6]
    set ::scoreboard(p2country) [lindex $sb 7]
    set ::scoreboard(p2score) [lindex $sb 8]
    set ::scoreboard(p2team) [lindex $sb 9]
    update_applied_scoreboard
}

proc applyscoreboard {} {
    set sb [ \
        ipc "applyscoreboard" \
        $::scoreboard(description) \
        $::scoreboard(subtitle) \
        $::scoreboard(stage) \
        $::scoreboard(p1name) \
        $::scoreboard(p1country) \
        $::scoreboard(p1score) \
        $::scoreboard(p1team) \
        $::scoreboard(p1character) \
        $::scoreboard(p2name) \
        $::scoreboard(p2country) \
        $::scoreboard(p2score) \
        $::scoreboard(p2team) \
        $::scoreboard(p2character) \
        $::scoreboard(c1title) \
        $::scoreboard(c1subtitle) \
        $::scoreboard(c2title) \
        $::scoreboard(c2subtitle) \
    ]
    update_applied_scoreboard
}

proc loadplayernames {} {
    set playernames [ipc "searchplayers" ""]
    .n.m.players.p1name configure -values $playernames
    .n.m.players.p2name configure -values $playernames
}

proc setupplayersuggestion {} {
    proc update_suggestions {_ key _} {
        if {!($key == "p1name" || $key == "p2name")} {
            return
        }
        set newvalue $::scoreboard($key)
        set widget .n.m.players.$key
        set matches [ipc "searchplayers" $newvalue]
        $widget configure -values $matches

        if {[llength matches] == 1 && [lindex $matches 0] == $newvalue} {
            set countryvar "p[string index $key 1]country"
            set country [lindex [ipc "getplayercountry" $newvalue] 0]
            set ::scoreboard($countryvar) $country
        }
    }
    trace add variable ::scoreboard write update_suggestions
}
proc setupcharacters {} {
    set widgetOne .n.m.players.p1character
    set widgetTwo .n.m.players.p2character
    set characters [ipc "loadcharacters"]
    $widgetOne configure -values $characters
    $widgetTwo configure -values $characters
}
proc setupstages {} {
    set widget .n.m.stage.entry
    set stages [ipc "loadstages"]
    $widget configure -values $stages
}
proc fetchplayers {} {
    if {$::startgg(token) == "" || $::startgg(slug) == ""} {
        set ::startgg(msg) "Please enter token & slug first."
        return
    }
    .n.s.buttons.fetch configure -state disabled
    .n.s.buttons.clear configure -state disabled
    .n.s.token configure -state disabled
    .n.s.tournamentslug configure -state disabled
    .n state disabled
    set ::startgg(msg) "Fetching..."
    ipc_write "fetchplayers" $::startgg(token) $::startgg(slug)
}

proc fetchplayers__resp {} {
    set resp [ipc_read]
    set status [lindex $resp 0]
    set msg [lindex $resp 1]

    set ::startgg(msg) $msg

    if {$status == "ok"} {
        loadplayernames
    }

    .n.s.buttons.fetch configure -state normal
    .n.s.buttons.clear configure -state normal
    .n.s.token configure -state normal
    .n.s.tournamentslug configure -state normal
    .n state !disabled
}

proc clearstartgg {} {
    set ::startgg(token) ""
    set ::startgg(slug) ""
    set ::startgg(msg) ""
    ipc_write "clearstartgg"
}
proc getstreamqueue {} {
    if {$::startgg(token) == "" || $::startgg(slug) == ""} {
        set ::startgg(msg) "Please enter token & slug first."
        return
    }
    .n.s.buttons.fetch configure -state disabled
    .n.s.buttons.clear configure -state disabled
    .n.s.token configure -state disabled
    .n.s.tournamentslug configure -state disabled
    set ::startgg(msg) "Fetching..."
    ipc_write "fetchlateststreamqueue" $::startgg(token) $::startgg(slug)
}
proc getstreamqueue__resp {} {
    set resp [ipc_read]
    set status [lindex $resp 0]
    set msg [lindex $resp 1]

    set ::startgg(msg) $msg

    if {$status == "ok"} {
        set ::scoreboard(p1name) [lindex $resp 2]
        set ::scoreboard(p1country) [lindex $resp 3]
        set ::scoreboard(p1score) [lindex $resp 4]
        set ::scoreboard(p1team) [lindex $resp 5]
        set ::scoreboard(p2name) [lindex $resp 6]
        set ::scoreboard(p2country) [lindex $resp 7]
        set ::scoreboard(p2score) [lindex $resp 8]
        set ::scoreboard(p2team) [lindex $resp 9]
    }

    .n.s.buttons.fetch configure -state normal
    .n.s.buttons.clear configure -state normal
    .n.s.token configure -state normal
    .n.s.tournamentslug configure -state normal
}

#TODO: Show bracket on frontend for editing/validation
proc getbracket {} {
    if {$::startgg(token) == "" || $::startgg(phasegroupid) == ""} {
        set ::startgg(msg) "Please enter token & phase group id first."
        return
    }
    .n.s.buttons.fetch configure -state disabled
    .n.s.buttons.clear configure -state disabled
    .n.s.token configure -state disabled
    .n.s.tournamentslug configure -state disabled
    set ::startgg(msg) "Fetching..."
    ipc_write "fetchbracket" $::startgg(token) $::startgg(phasegroupid)
}
proc getbracket__resp {} {
    set resp [ipc_read]
    set status [lindex $resp 0]
    set msg [lindex $resp 1]

    set ::startgg(msg) $msg

    .n.s.buttons.fetch configure -state normal
    .n.s.buttons.clear configure -state normal
    .n.s.token configure -state normal
    .n.s.tournamentslug configure -state normal
}

proc discardscoreboard {} {
    foreach key [array names ::scoreboard] {
        set ::scoreboard($key) $::applied_scoreboard($key)
    }
    # Country is updated whenever player name is updated,
    # so make sure we set countries last.
    set ::scoreboard(p1country) $::applied_scoreboard(p1country)
    set ::scoreboard(p2country) $::applied_scoreboard(p2country)
}

proc update_applied_scoreboard {} {
    foreach key [array names ::scoreboard] {
        set ::applied_scoreboard($key) $::scoreboard($key)
    }
}

proc setupdiffcheck {} {
    # Define styling for "dirty"
    foreach x {TEntry TCombobox TSpinbox} {
        ttk::style configure "Dirty.$x" -fieldbackground #dffcde
    }

    trace add variable ::scoreboard write ::checkdiff
    trace add variable ::applied_scoreboard write ::checkdiff
}

proc checkdiff {_ key _} {
    set widget $::var_to_widget($key)
    if {$::scoreboard($key) == $::applied_scoreboard($key)} {
        $widget configure -style [winfo class $widget]
    } else {
        $widget configure -style "Dirty.[winfo class $widget]"
    }
}
