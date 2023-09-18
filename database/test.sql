select contributor_name, pullreq_url, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as latest_points
		from contributor_record_models as t1
		where contributor_name = "Smuzzy-waiii"
		GROUP by pullreq_url, contributor_name

select contributor_name, maintainer_name, pullreq_url, (SELECT points_allotted FROM contributor_record_models where t1.pullreq_url = pullreq_url order by created_at desc limit 1) as points_allotted
    from contributor_record_models as t1
    where contributor_name = "sid-008"
    GROUP by pullreq_url, contributor_name;

select contributor_name, maintainer_name, pullreq_url, points_allotted from contributor_record_models
GROUP by pullreq_url, contributor_name
order by created_at desc limit 1

SELECT crm.contributor_name, crm.maintainer_name, crm.pullreq_url, crm.points_allotted
FROM contributor_record_models as crm
         JOIN (
    SELECT MAX(id) as max_id
    FROM contributor_record_models
    WHERE contributor_name = "sid-008"
    GROUP BY pullreq_url
) AS subq ON crm.id = subq.max_id;