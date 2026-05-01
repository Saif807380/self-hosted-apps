#!/usr/bin/env python3
import os
import glob
from mutagen.oggopus import OggOpus

# The folder where your un-imported yt-dlp files are sitting
DIRECTORY = os.path.expanduser('~/Music/')

print("Starting tag update...")

# Find all .opus files in the directory
for filepath in glob.glob(os.path.join(DIRECTORY, '*.opus')):
    try:
        audio = OggOpus(filepath)
        
        force_artist = os.environ.get('FORCE_ARTIST')
        force_title = os.environ.get('FORCE_TITLE')
        force_album = os.environ.get('FORCE_ALBUM')

        if force_artist:
            audio['artist'] = force_artist
            audio['albumartist'] = force_artist.split(',')[0].strip()
        elif 'artist' in audio:
            # Grab the artist string, split it by the comma, and take the first item
            artist_string = audio['artist'][0]
            audio['albumartist'] = artist_string.split(',')[0].strip()

        if force_title:
            audio['title'] = force_title

        if force_album:
            audio['album'] = force_album

        audio.save()
        print(f"✅ Updated: {os.path.basename(filepath)} | Artist -> {audio.get('artist', [''])} | Title -> {audio.get('title', [''])}")
            
    except Exception as e:
        print(f"❌ Error processing {filepath}: {e}")

print("Done!")