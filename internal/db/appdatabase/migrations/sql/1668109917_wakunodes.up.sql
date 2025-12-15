INSERT INTO cluster_nodes(node, type)
SELECT node, 'waku' 
FROM cluster_nodes
WHERE type = "relay"
