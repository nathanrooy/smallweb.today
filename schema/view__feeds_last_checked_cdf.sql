create or replace view feeds_last_checked_cdf as
with last_checked as (
	select
		feed_url,
		coalesce(last_checked, '1970-01-01 00:00:00+00') as last_checked
	from feeds_filtered as f
	left join feed_states as s using(feed_url)
),

check_deltas as (
	select
		feed_url,
		floor(extract(epoch from (now() - last_checked)) / 60)::int as minutes_ago
	from last_checked
),

frequency_counts as (
    select
        minutes_ago,
        count(*) as feed_count
    from  check_deltas
    group by minutes_ago
),

cdf as (
    select
        minutes_ago,
        (sum(feed_count) over (order by minutes_ago asc) * 100.0 / sum(feed_count) over ()) as cumsum_norm
    from frequency_counts
)

select
    minutes_ago,
    round(cumsum_norm, 2) as cdf
from cdf
order by minutes_ago asc;