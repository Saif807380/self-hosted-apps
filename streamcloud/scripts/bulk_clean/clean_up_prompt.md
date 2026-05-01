Act as a precise music archivist. I have a directory called ./llm_inputs/ containing multiple JSON files. Each file contains an array of file objects from my music library.

Your task is to process EVERY file in ./llm_inputs/ and create a corresponding output file in a directory called ./llm_outputs/ (e.g., batch_001.json -> ./llm_outputs/batch_001.json).

Processing Rules for each track:

0. **Stay Focused**: Use research tools ONLY if the metadata is not obvious or needs verification. Do not perform broad codebase searches.
1. **Identify Metadata**: Find the true primary Artist, original Studio Album (ignore compilations, "Greatest Hits", or playlists), and correct Track Name. 
2. **Search**: Search the internet to find the correct album and artist names for the original track name.
3. **Collaborations**: For collaborations (e.g., "Artist A & Artist B"), assign the track to the primary artist (Artist A). Do not create joint artist folders. 
4. **Primary Artist**: If multiple artists are listed, select only the first one for the "artist" and "albumartist" fields.
5. **Clean Tags**: Strip out junk text like "(Official Audio)", "(Lyric Video)", "[HD]", or random numbers from titles.
6. **New Path**: The new_path format MUST be: /Primary Artist/Album Name/Track Name.extension. Retain the original file extension.

Output Format:
For each file, output a strict JSON array containing mapping objects. Do NOT use markdown code blocks inside the file. Write pure JSON to the files.

Schema:
```json
[
  {
    "path": "original_absolute_path",
    "title": "Correct Track Title",
    "album": "Original Studio Album",
    "artist": "Primary Artist",
    "albumartist": "Primary Artist",
    "new_path": "/Artist/Album/Track.extension"
  }
]
```
