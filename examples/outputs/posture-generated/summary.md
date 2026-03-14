# Sheaft Report Summary

- Decision: **warn**
- Mode: `warn`
- Overall availability: `0.9609`
- Weighted overall availability: `0.9596`
- Cross-profile availability: `0.7681`
- Cross-profile weighted availability: `0.7249`
- Risk score: `0.0404`
- Confidence: `0.94`

## Profiles

- `steady-state`: decision=`warn`, weighted=`0.9596`, unweighted=`0.9609`, below-threshold=`2`
- `service-fault`: decision=`warn`, weighted=`0.8161`, unweighted=`0.8448`, below-threshold=`3`
- `fixed-blast-radius`: decision=`warn`, weighted=`0.3990`, unweighted=`0.4986`, below-threshold=`3`

## Endpoint results

- `steady-state` / `checkout:POST /process`: availability=`0.9716`, threshold=`0.9700`, status=`pass`
- `steady-state` / `frontend:GET /checkout`: availability=`0.9417`, threshold=`0.9850`, status=`warn`
- `steady-state` / `frontend:GET /health`: availability=`0.9695`, threshold=`0.9850`, status=`warn`

## Diffs

- Baseline `last-release`: weighted delta=`-0.1806`, unweighted delta=`-0.1374`
