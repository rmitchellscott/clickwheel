#!/usr/bin/env python3
"""Generate golden iTunesDB files using iOpenPod as reference implementation.

Run from the clickwheel repo root:
    python3 internal/ipod/itunesdb/testdata/generate_golden.py

Requires iOpenPod to be checked out at ~/github/TheRealSavi/iOpenPod.
"""

import json
import os
import struct
import sys

IOPENPOD_PATH = os.path.expanduser("~/github/TheRealSavi/iOpenPod")
sys.path.insert(0, IOPENPOD_PATH)

from iTunesDB_Writer.mhbd_writer import write_mhbd
from iTunesDB_Writer.mhit_writer import TrackInfo
from iTunesDB_Writer.mhyp_writer import PlaylistInfo
from iTunesDB_Writer.mhod_spl_writer import (
    SmartPlaylistPrefs,
    SmartPlaylistRule,
    SmartPlaylistRules,
)
from ipod_models import DeviceCapabilities, ChecksumType

OUTDIR = os.path.dirname(os.path.abspath(__file__))

# Device profiles representing distinct capability combinations.
# We skip hash-requiring profiles (Hash58/72/AB) since those need
# FirewireIDs and produce device-specific output. We test the
# structural output for the 4 non-hash profiles.
DEVICE_PROFILES = {
    "ipod1g": DeviceCapabilities(
        supports_podcast=False,
        supports_video=False,
        supports_gapless=False,
        db_version=0x13,
    ),
    "ipod4g": DeviceCapabilities(
        supports_podcast=True,
        supports_video=False,
        supports_gapless=False,
        db_version=0x13,
    ),
    "ipodvideo5g": DeviceCapabilities(
        supports_podcast=True,
        supports_video=True,
        supports_gapless=True,
        db_version=0x19,
    ),
    "classic": DeviceCapabilities(
        supports_podcast=True,
        supports_video=True,
        supports_gapless=True,
        db_version=0x30,
    ),
    "none": None,  # No capabilities (default path)
}


def music_tracks():
    """3 tracks, 1 album, 1 artist."""
    return [
        TrackInfo(
            title="Track One",
            location=":iPod_Control:Music:F00:AAAA.mp3",
            artist="Artist A",
            album="Album X",
            genre="Rock",
            size=5000000,
            length=240000,
            bitrate=320,
            sample_rate=44100,
            filetype="mp3",
            year=2020,
            track_number=1,
            total_tracks=3,
            media_type=0x01,
        ),
        TrackInfo(
            title="Track Two",
            location=":iPod_Control:Music:F01:BBBB.mp3",
            artist="Artist A",
            album="Album X",
            genre="Rock",
            size=4500000,
            length=200000,
            bitrate=320,
            sample_rate=44100,
            filetype="mp3",
            year=2020,
            track_number=2,
            total_tracks=3,
            media_type=0x01,
        ),
        TrackInfo(
            title="Track Three",
            location=":iPod_Control:Music:F02:CCCC.mp3",
            artist="Artist A",
            album="Album X",
            genre="Rock",
            size=4800000,
            length=220000,
            bitrate=320,
            sample_rate=44100,
            filetype="mp3",
            year=2020,
            track_number=3,
            total_tracks=3,
            media_type=0x01,
        ),
    ]


def multi_album_tracks():
    """6 tracks across 3 albums, 3 artists."""
    return [
        TrackInfo(
            title="Alpha",
            location=":iPod_Control:Music:F00:AAA1.mp3",
            artist="Band One",
            album="First Album",
            genre="Pop",
            size=5000000,
            length=200000,
            bitrate=256,
            sample_rate=44100,
            filetype="mp3",
            year=2018,
            track_number=1,
            total_tracks=2,
            media_type=0x01,
        ),
        TrackInfo(
            title="Beta",
            location=":iPod_Control:Music:F01:AAA2.mp3",
            artist="Band One",
            album="First Album",
            genre="Pop",
            size=4800000,
            length=190000,
            bitrate=256,
            sample_rate=44100,
            filetype="mp3",
            year=2018,
            track_number=2,
            total_tracks=2,
            media_type=0x01,
        ),
        TrackInfo(
            title="Gamma",
            location=":iPod_Control:Music:F02:BBB1.mp3",
            artist="Singer Two",
            album="Second Record",
            genre="Jazz",
            size=6000000,
            length=300000,
            bitrate=320,
            sample_rate=48000,
            filetype="m4a",
            year=2019,
            track_number=1,
            total_tracks=2,
            media_type=0x01,
        ),
        TrackInfo(
            title="Delta",
            location=":iPod_Control:Music:F03:BBB2.mp3",
            artist="Singer Two",
            album="Second Record",
            genre="Jazz",
            size=5500000,
            length=280000,
            bitrate=320,
            sample_rate=48000,
            filetype="m4a",
            year=2019,
            track_number=2,
            total_tracks=2,
            media_type=0x01,
        ),
        TrackInfo(
            title="Epsilon",
            location=":iPod_Control:Music:F04:CCC1.mp3",
            artist="DJ Three",
            album="Third Mix",
            genre="Electronic",
            size=7000000,
            length=360000,
            bitrate=320,
            sample_rate=44100,
            filetype="mp3",
            year=2021,
            track_number=1,
            total_tracks=2,
            media_type=0x01,
        ),
        TrackInfo(
            title="Zeta",
            location=":iPod_Control:Music:F05:CCC2.mp3",
            artist="DJ Three",
            album="Third Mix",
            genre="Electronic",
            size=6500000,
            length=340000,
            bitrate=320,
            sample_rate=44100,
            filetype="mp3",
            year=2021,
            track_number=2,
            total_tracks=2,
            media_type=0x01,
        ),
    ]


def audiobook_tracks():
    """1 audiobook with bookmark/position flags."""
    return [
        TrackInfo(
            title="Chapter 1 - The Beginning",
            location=":iPod_Control:Music:F00:BOOK.m4b",
            artist="Author Name",
            album="My Audiobook",
            size=50000000,
            length=3600000,
            filetype="m4b",
            media_type=0x08,
            remember_position=True,
            skip_when_shuffling=True,
            bookmark_time=1200000,
        ),
    ]


def podcast_tracks():
    """2 podcast episodes with podcast flags and URLs."""
    return [
        TrackInfo(
            title="Episode 1: Pilot",
            location=":iPod_Control:Music:F00:POD1.mp3",
            artist="Podcast Host",
            album="My Podcast",
            size=30000000,
            length=1800000,
            bitrate=128,
            sample_rate=44100,
            filetype="mp3",
            media_type=0x04,
            podcast_flag=1,
            remember_position=True,
            skip_when_shuffling=True,
            podcast_enclosure_url="https://example.com/ep1.mp3",
            podcast_rss_url="https://example.com/feed.xml",
            category="Technology",
        ),
        TrackInfo(
            title="Episode 2: Follow Up",
            location=":iPod_Control:Music:F01:POD2.mp3",
            artist="Podcast Host",
            album="My Podcast",
            size=35000000,
            length=2100000,
            bitrate=128,
            sample_rate=44100,
            filetype="mp3",
            media_type=0x04,
            podcast_flag=1,
            remember_position=True,
            skip_when_shuffling=True,
            podcast_enclosure_url="https://example.com/ep2.mp3",
            podcast_rss_url="https://example.com/feed.xml",
            category="Technology",
        ),
    ]


def playlist_scenario():
    """Music tracks + 2 user playlists. Returns (tracks, playlists)."""
    tracks = multi_album_tracks()
    # Playlists reference tracks by dbid. Since we don't know dbids yet,
    # we set them explicitly and reference those.
    for i, t in enumerate(tracks):
        t.dbid = 1000 + i

    pl1 = PlaylistInfo(
        name="Favorites",
        track_ids=[1000, 1002, 1004],  # Alpha, Gamma, Epsilon
    )
    pl2 = PlaylistInfo(
        name="Chill",
        track_ids=[1001, 1003],  # Beta, Delta
    )
    return tracks, [pl1, pl2]


def smart_playlist_scenario():
    """Music tracks + 1 smart playlist filtering by genre."""
    tracks = multi_album_tracks()
    for i, t in enumerate(tracks):
        t.dbid = 2000 + i

    prefs = SmartPlaylistPrefs(
        live_update=True,
        check_rules=True,
        check_limits=False,
        limit_type=0x03,
        limit_sort=0x02,
        limit_value=25,
    )
    rules = SmartPlaylistRules(
        conjunction="AND",
        rules=[
            SmartPlaylistRule(
                field_id=0x08,  # Genre
                action_id=0x01000002,  # contains
                string_value="Rock",
            ),
        ],
    )
    spl = PlaylistInfo(
        name="Rock Songs",
        track_ids=[2000, 2001],  # Alpha, Beta (genre=Pop, but rule is illustrative)
        smart_prefs=prefs,
        smart_rules=rules,
    )
    return tracks, [spl]


def mixed_tracks():
    """Music + audiobook + podcast."""
    m = music_tracks()[:2]
    a = audiobook_tracks()
    p = podcast_tracks()[:1]
    return m + a + p


def compilation_tracks():
    """Compilation album with multiple artists."""
    artists = ["Artist X", "Artist Y", "Artist Z"]
    tracks = []
    for i, art in enumerate(artists):
        tracks.append(
            TrackInfo(
                title=f"Comp Track {i+1}",
                location=f":iPod_Control:Music:F0{i}:COMP{i}.mp3",
                artist=art,
                album="Various Artists Compilation",
                album_artist="Various Artists",
                genre="Pop",
                size=4000000,
                length=180000,
                bitrate=256,
                sample_rate=44100,
                filetype="mp3",
                year=2022,
                track_number=i + 1,
                total_tracks=3,
                compilation=True,
                media_type=0x01,
            )
        )
    return tracks


def unicode_tracks():
    """International characters in metadata."""
    return [
        TrackInfo(
            title="Für Elise",
            location=":iPod_Control:Music:F00:UNIC.mp3",
            artist="ベートーヴェン",
            album="Classique",
            genre="Classique",
            size=3000000,
            length=180000,
            bitrate=256,
            sample_rate=44100,
            filetype="mp3",
            year=1810,
            media_type=0x01,
        ),
        TrackInfo(
            title="Ça Plane Pour Moi",
            location=":iPod_Control:Music:F01:FREN.mp3",
            artist="Plastic Bertrand",
            album="Ça Plane Pour Moi",
            genre="Punk",
            size=3500000,
            length=190000,
            bitrate=256,
            sample_rate=44100,
            filetype="mp3",
            year=1977,
            media_type=0x01,
        ),
    ]


# ── Scenario definitions ──────────────────────────────────────────────

SCENARIOS = {}


def _add(name, tracks, playlists=None):
    SCENARIOS[name] = (tracks, playlists or [])


_add("basic_music", music_tracks())
_add("multi_album", multi_album_tracks())
_add("audiobook", audiobook_tracks())
_add("podcast", podcast_tracks())
_add("compilation", compilation_tracks())
_add("unicode", unicode_tracks())
_add("mixed", mixed_tracks())
_add("empty", [])

pl_tracks, pl_playlists = playlist_scenario()
_add("playlist", pl_tracks, pl_playlists)

spl_tracks, spl_playlists = smart_playlist_scenario()
_add("smart_playlist", spl_tracks, spl_playlists)


def extract_structure(data):
    """Extract structural summary from raw iTunesDB bytes for JSON metadata."""
    if len(data) < 244 or data[:4] != b"mhbd":
        return None

    info = {
        "size": len(data),
        "version": struct.unpack_from("<I", data, 0x10)[0],
        "num_datasets": struct.unpack_from("<I", data, 0x14)[0],
        "datasets": [],
    }

    hdr_len = struct.unpack_from("<I", data, 4)[0]
    num_ds = info["num_datasets"]
    pos = hdr_len

    for _ in range(num_ds):
        if pos + 16 > len(data) or data[pos : pos + 4] != b"mhsd":
            break

        ds_total = struct.unpack_from("<I", data, pos + 8)[0]
        ds_type = struct.unpack_from("<I", data, pos + 12)[0]
        ds_hdr = struct.unpack_from("<I", data, pos + 4)[0]

        child_pos = pos + ds_hdr
        ds_info = {"type": ds_type, "total_len": ds_total}

        if child_pos + 12 <= len(data):
            child_magic = data[child_pos : child_pos + 4].decode("ascii", errors="replace")
            child_count = struct.unpack_from("<I", data, child_pos + 8)[0]
            ds_info["child_magic"] = child_magic
            ds_info["child_count"] = child_count

            if child_magic == "mhlt":
                ds_info["tracks"] = extract_tracks(data, child_pos, pos + ds_total)
            elif child_magic == "mhlp":
                ds_info["playlists"] = extract_playlists(data, child_pos, pos + ds_total)

        info["datasets"].append(ds_info)
        pos += ds_total

    return info


def extract_tracks(data, mhlt_pos, ds_end):
    hdr_len = struct.unpack_from("<I", data, mhlt_pos + 4)[0]
    count = struct.unpack_from("<I", data, mhlt_pos + 8)[0]
    tracks = []
    pos = mhlt_pos + hdr_len

    for _ in range(count):
        if pos + 16 > ds_end or data[pos : pos + 4] != b"mhit":
            break
        total_len = struct.unpack_from("<I", data, pos + 8)[0]
        track_id = struct.unpack_from("<I", data, pos + 0x10)[0]
        mhod_count = struct.unpack_from("<I", data, pos + 0x0C)[0]

        mhod_types = []
        mpos = pos + struct.unpack_from("<I", data, pos + 4)[0]
        for _ in range(mhod_count):
            if mpos + 16 > pos + total_len or data[mpos : mpos + 4] != b"mhod":
                break
            mt = struct.unpack_from("<I", data, mpos + 12)[0]
            mhod_types.append(mt)
            mpos += struct.unpack_from("<I", data, mpos + 8)[0]

        tracks.append({
            "track_id": track_id,
            "mhod_count": mhod_count,
            "mhod_types": mhod_types,
        })
        pos += total_len

    return tracks


def extract_playlists(data, mhlp_pos, ds_end):
    hdr_len = struct.unpack_from("<I", data, mhlp_pos + 4)[0]
    count = struct.unpack_from("<I", data, mhlp_pos + 8)[0]
    playlists = []
    pos = mhlp_pos + hdr_len

    for _ in range(count):
        if pos + 24 > ds_end or data[pos : pos + 4] != b"mhyp":
            break
        total_len = struct.unpack_from("<I", data, pos + 8)[0]
        mhod_count = struct.unpack_from("<I", data, pos + 0x0C)[0]
        item_count = struct.unpack_from("<I", data, pos + 0x10)[0]
        is_master = struct.unpack_from("<I", data, pos + 0x14)[0]

        playlists.append({
            "mhod_count": mhod_count,
            "item_count": item_count,
            "is_master": is_master,
        })
        pos += total_len

    return playlists


def generate(scenario_name, tracks, playlists, profile_name, caps):
    """Generate one golden file + metadata JSON."""
    kwargs = dict(tracks=tracks, capabilities=caps)
    if playlists:
        kwargs["playlists_type2"] = playlists

    data = write_mhbd(**kwargs)
    structure = extract_structure(data)

    basename = f"golden_{scenario_name}_{profile_name}"
    bin_path = os.path.join(OUTDIR, f"{basename}.bin")
    json_path = os.path.join(OUTDIR, f"{basename}.json")

    with open(bin_path, "wb") as f:
        f.write(data)
    with open(json_path, "w") as f:
        json.dump(structure, f, indent=2)

    print(f"  {basename}: {len(data)} bytes, {structure['num_datasets']} datasets")


def main():
    # For each scenario × profile, only generate the combos that make sense.
    # Skip podcast scenarios on devices without podcast support.
    # Skip hash profiles (tested separately since they need FirewireIDs).

    for profile_name, caps in DEVICE_PROFILES.items():
        print(f"\n=== Profile: {profile_name} ===")

        supports_podcast = caps is None or caps.supports_podcast

        for scenario_name, (tracks, playlists) in SCENARIOS.items():
            if scenario_name in ("podcast", "mixed") and not supports_podcast:
                continue

            generate(scenario_name, list(tracks), list(playlists), profile_name, caps)


if __name__ == "__main__":
    main()
