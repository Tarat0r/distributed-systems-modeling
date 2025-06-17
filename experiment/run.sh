#!/bin/bash

# Path to your executable (adjust if needed)
PROGRAM="./main"

# Create output directory if needed
mkdir -p output_logs

# Experiment counter
EXPERIMENT_ID=0
export LC_NUMERIC=C

# Loop over different parameter combinations
for NODES in {100..500..25}; do
  for FANOUT in {1..5}; do
    for DELAY in {10..30..10}; do
      for ALIVE in $(seq 0.6 0.1 1.0); do
        for LOSS in $(seq 0.0 0.05 0.3); do
          for CORRUPT in $(seq 0.0 0.05 0.3); do

            echo "Running experiment ID $EXPERIMENT_ID with Parameters: NODES=$NODES, FANOUT=$FANOUT, DELAY=$DELAY, ALIVE=$ALIVE, LOSS=$LOSS, CORRUPT=$CORRUPT" >> "output_logs/experiment_progress.log"
            $PROGRAM \
              -id "$EXPERIMENT_ID" \
              -nodes "$NODES" \
              -domains "$(( FANOUT + 1 ))" \
              -fanout "$FANOUT" \
              -delay "$DELAY" \
              -alive "$ALIVE" \
              -loss "$LOSS" \
              -corrupt "$CORRUPT" \
              -timer "$(( NODES / 2 ))" \
              > "output_logs/exp_${EXPERIMENT_ID}.log" 2>&1

            ((EXPERIMENT_ID++))

          done
        done
      done
    done
  done
done

echo "✅ All experiments completed."
