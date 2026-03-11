drop view if exists feeds_queue;
create view if not exists feeds_queue as
select
	base_url,
	feed_url,
	coalesce(last_checked, '1970-01-01 00:00:00+00'::timestamptz) as last_checked,
	coalesce(last_modified, '') as last_modified,
	coalesce(etag, '') as etag
from feeds_filtered as f
left join feed_states as s using(feed_url)
where
	(
		s.last_checked is null

		-- only feeds that have not been checked within the last hour are eligible
		or s.last_checked < (now() - interval '1 hour')
	)