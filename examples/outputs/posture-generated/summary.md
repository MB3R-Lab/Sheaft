# Sheaft Report Summary

- Decision: **warn**
- Mode: `warn`
- Overall availability: `0.9559`
- Weighted overall availability: `0.9500`
- Cross-profile availability: `0.8184`
- Cross-profile weighted availability: `0.7944`
- Risk score: `0.0500`
- Confidence: `0.72`

## Profiles

- `steady-state`: decision=`warn`, weighted=`0.9500`, unweighted=`0.9559`, below-threshold=`2`
- `service-fault`: decision=`warn`, weighted=`0.8599`, unweighted=`0.8761`, below-threshold=`2`
- `fixed-blast-radius`: decision=`warn`, weighted=`0.5733`, unweighted=`0.6232`, below-threshold=`2`

## Endpoint results

- `steady-state` / `frontend:GET /checkout`: availability=`0.9413`, threshold=`0.9700`, status=`warn`
- `steady-state` / `frontend:GET /health`: availability=`0.9705`, threshold=`0.9850`, status=`warn`

## Diffs

- Baseline `last-release`: weighted delta=`-0.1111`, unweighted delta=`-0.0871`
