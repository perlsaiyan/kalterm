# Kalterm
[![GitHub Super-Linter](https://github.com/perlsaiyan/kalterm/workflows/Lint/badge.svg)](https://github.com/marketplace/actions/super-linter)
[![Build Check](https://github.com/perlsaiyan/kalterm/workflows/golang-pipline/badge.svg)](https://github.com/github/docs/actions/workflows/main.yml/badge.svg)

A simple visualization from the terminal of tintin++ bot status.

It uses kalterm.tin (in the tintin directory) to create a #port
session on 9595, after which the kalterm cli app can connect and begin
receiving data.  The graphs are pre-defined for now.  The Log window can be
populated by running `kalterm_msg <log message to send>` in tintin++.

![Screenshot](/screenshots/example.gif)

The dark blue line in the line charts are projections for hourly rates
based on previous one minute interval.  The white line represents 
actual gains over the previous hour.

The pie chart is progression along the current #path.  In the example,
we're always at step one with a shrinking path size because of how
the bot works in that particular area (remapping path from every step,
because mobs can move the player off path).

# Links
[Tintin++](https://tintin.mudhalla.net)

[Golang](https://go.dev)

