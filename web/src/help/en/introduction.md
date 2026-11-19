# Introduction

Moji is a self-hosted companion for Stash. It coordinates film discovery, resource selection, qBittorrent downloads, file delivery, and Stash scanning, generation, identification and verification.

## Data and responsibilities

Stash stores performers, subscription markers and the media library; StashBox supplies film metadata. Moji stores check schedules, release decisions, download tasks and recovery state. qBittorrent handles BT transfers. Built-in indexers and optional Jackett provide resource searches.

## Main workflow

1. Find and confirm a StashBox film in **Discover** or performer details.
2. Create an automatic sourcing task, or use a specified resource after confirming the film.
3. Let Moji validate resources and monitor qBittorrent; uncertain files wait for your selection.
4. Copy or move a single movie or its assembled parts into the library.
5. Let Stash scan, generate and identify it. Moji verifies metadata and the exact StashBox association before completing the task.

Download or scan completion alone does not complete the task. Blocked tasks retain their cause and can resume from the affected step after it is resolved.
