from dataclasses import dataclass
from datetime import datetime, timezone, timedelta
from time import mktime

import feedparser

RSS_FEEDS = [
    ("Formula 1 Official", "https://www.formula1.com/en/latest/all.xml"),
    ("Motorsport.com", "https://www.motorsport.com/rss/f1/news/"),
    ("Google News F1", "https://news.google.com/rss/search?q=formula+1&hl=en"),
]

MAX_ARTICLES = 30
SIMILARITY_THRESHOLD = 0.6


@dataclass
class NewsItem:
    title: str
    summary: str
    source: str
    published: datetime
    link: str


def _parse_published(entry) -> datetime | None:
    published = entry.get("published_parsed") or entry.get("updated_parsed")
    if published is None:
        return None
    try:
        return datetime.fromtimestamp(mktime(published), tz=timezone.utc)
    except Exception:
        return None


def _title_similarity(a: str, b: str) -> float:
    words_a = set(a.lower().split())
    words_b = set(b.lower().split())
    if not words_a or not words_b:
        return 0.0
    intersection = words_a & words_b
    return len(intersection) / min(len(words_a), len(words_b))


def _deduplicate(items: list[NewsItem]) -> list[NewsItem]:
    unique = []
    for item in items:
        is_dup = any(
            _title_similarity(item.title, existing.title) > SIMILARITY_THRESHOLD
            for existing in unique
        )
        if not is_dup:
            unique.append(item)
    return unique


def get_f1_news(days_back: int = 5) -> list[NewsItem]:
    cutoff = datetime.now(tz=timezone.utc) - timedelta(days=days_back)
    all_items: list[NewsItem] = []

    for source_name, feed_url in RSS_FEEDS:
        try:
            feed = feedparser.parse(feed_url)
        except Exception:
            continue

        for entry in feed.entries:
            published = _parse_published(entry)
            if published is None or published < cutoff:
                continue

            summary = entry.get("summary", "")
            # Strip HTML tags from summary
            if "<" in summary:
                import re
                summary = re.sub(r"<[^>]+>", "", summary).strip()

            all_items.append(NewsItem(
                title=entry.get("title", "").strip(),
                summary=summary[:500],
                source=source_name,
                published=published,
                link=entry.get("link", ""),
            ))

    all_items.sort(key=lambda n: n.published, reverse=True)
    all_items = _deduplicate(all_items)
    return all_items[:MAX_ARTICLES]
