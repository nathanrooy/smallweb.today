drop materialized view if exists feeds_filtered;
create materialized view feeds_filtered as
with feeds as (
    select
  		max(base_url) as base_url,
        feed_url,
        array_agg(source) as sources,
        coalesce(max(trusted), false) as verified,
		coalesce(sum(trusted::int), 0) as verified_count
    from feeds
    left join feed_sources using(source)
    group by feed_url
),

feed_annotations_aggregated as (
	select
		base_url,
		max(
			case
				when annotation_type = 'not_small_web'
				then 1
				else 0
			end
		)::bool as not_small_web,
		max(
			case
				when annotation_type = 'is_small_web'
				then 1
				else 0
			end
		)::bool as is_small_web
	from post_annotations
	group by base_url
),

final as (
    select
    	feeds.base_url,
    	feeds.feed_url,
    	case
			when is_small_web is true
			then array_append(feeds.sources, 'admin')
			else feeds.sources
		end as sources,
		case
			when is_small_web is true
			then true
			else feeds.verified
		end as verified,
		case
			when is_small_web is true
			then verified_count + 1
			else verified_count
		end as verified_count,
		case
			when is_small_web is true
			then false
			else coalesce(not_small_web, false)
		end as not_small_web
    from feeds
    left join feed_annotations_aggregated using(base_url)
)

select
	base_url,
	feed_url,
	sources,
	verified,
	verified_count
from final
where not_small_web = false