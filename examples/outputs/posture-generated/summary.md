# Sheaft Report Summary

- Decision: **fail**
- Mode: `fail`
- Overall availability: `0.9547`
- Weighted overall availability: `0.9600`
- Cross-profile availability: `0.7114`
- Cross-profile weighted availability: `0.8589`
- Risk score: `0.0400`
- Confidence: `0.95`

## Profiles

- `steady-state`: decision=`fail`, weighted=`0.9600`, unweighted=`0.9547`, below-threshold=`2`
- `az-us-east-1a-outage`: decision=`fail`, weighted=`0.9323`, unweighted=`0.9266`, below-threshold=`2`
- `payment-brownout`: decision=`fail`, weighted=`0.7699`, unweighted=`0.4812`, below-threshold=`2`
- `checkout-payment-cut`: decision=`fail`, weighted=`0.7732`, unweighted=`0.4833`, below-threshold=`2`

## Why

- `endpoint_below_threshold` profile=`steady-state` endpoint=`gateway:GET /explicit` delta=`-0.0441`: endpoint "gateway:GET /explicit" availability 0.9459 is below threshold 0.9900
- `endpoint_below_threshold` profile=`steady-state` endpoint=`gateway:POST /checkout` delta=`-0.0265`: endpoint "gateway:POST /checkout" availability 0.9635 is below threshold 0.9900
- `endpoint_below_threshold` profile=`az-us-east-1a-outage` endpoint=`gateway:GET /explicit` delta=`-0.0729`: endpoint "gateway:GET /explicit" availability 0.9171 is below threshold 0.9900
- `endpoint_below_threshold` profile=`az-us-east-1a-outage` endpoint=`gateway:POST /checkout` delta=`-0.0539`: endpoint "gateway:POST /checkout" availability 0.9361 is below threshold 0.9900
- `endpoint_below_threshold` profile=`payment-brownout` endpoint=`gateway:GET /explicit` delta=`-0.9900`: endpoint "gateway:GET /explicit" availability 0.0000 is below threshold 0.9900
- `endpoint_below_threshold` profile=`payment-brownout` endpoint=`gateway:POST /checkout` delta=`-0.0276`: endpoint "gateway:POST /checkout" availability 0.9624 is below threshold 0.9900
- `assertion_failed` profile=`payment-brownout`: expected_success_rate path >= 0.1000 (actual=0.0000)
- `endpoint_below_threshold` profile=`checkout-payment-cut` endpoint=`gateway:GET /explicit` delta=`-0.9900`: endpoint "gateway:GET /explicit" availability 0.0000 is below threshold 0.9900
- `endpoint_below_threshold` profile=`checkout-payment-cut` endpoint=`gateway:POST /checkout` delta=`-0.0235`: endpoint "gateway:POST /checkout" availability 0.9665 is below threshold 0.9900

## Endpoint results

- `steady-state` / `gateway:GET /explicit`: availability=`0.9459`, threshold=`0.9900`, delta=`-0.0441`, status=`fail`
- `steady-state` / `gateway:POST /checkout`: availability=`0.9635`, threshold=`0.9900`, delta=`-0.0265`, status=`fail`

## Diffs

- Baseline `bering-1.3.0-baseline`: weighted delta=`0.0000`, unweighted delta=`0.0000`
