#!/usr/bin/env python3
import os
import glob
from mutagen.oggopus import OggOpus

# The folder where your un-imported yt-dlp files are sitting
# DIRECTORY = '/srv/media/staging/'
DIRECTORY = '/home/saifkazi/Music/'

print("Starting tag update...")

# Find all .opus files in the directory
for filepath in glob.glob(os.path.join(DIRECTORY, '*.opus')):
    try:
        audio = OggOpus(filepath)
        
        # Check if the 'artist' tag exists
        if 'artist' in audio:
            # Grab the artist string, split it by the comma, and take the first item
            artist_string = audio['artist'][0]
            first_artist = artist_string.split(',')[0].strip()
            
            # Write it to the 'albumartist' tag
            audio['albumartist'] = first_artist
            audio.save()
            
            print(f"✅ Updated: {os.path.basename(filepath)} | Album Artist -> {first_artist}")
            
    except Exception as e:
        print(f"❌ Error processing {filepath}: {e}")

print("Done!")