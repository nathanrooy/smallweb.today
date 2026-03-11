create or replace view posts_latest_filtered as
select
	base_url,
	feed_url,
	post_url,
	post_title,
	utc_discovered,
	utc_published
from posts_latest_annotated
where
	hide = false
	and verified_count > 0
	and utc_discovered > now() - interval '24 hours'
 order by utc_discovered desc