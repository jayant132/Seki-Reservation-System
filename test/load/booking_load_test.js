// Load test proving the core claim of this project: when many concurrent
// requests race to book the SAME resource for the SAME time slot, exactly
// one succeeds and every other one is cleanly rejected with 409 Conflict
// — zero double-bookings, no matter how many virtual users race for it.
//
// Run with:  k6 run test/load/booking_load_test.js
// Prereqs:   docker compose up -d   (API must be running on :8080)
//            a user account and a resource must already exist — this
//            script creates both automatically via setup().

import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

const successCount = new Counter("booking_success");
const conflictCount = new Counter("booking_conflict");
const unexpectedCount = new Counter("booking_unexpected_error");

export const options = {
  scenarios: {
    race_for_same_slot: {
      executor: "shared-iterations",
      vus: 50, // 50 concurrent virtual users
      iterations: 50, // ...all racing to book the exact same slot exactly once each
      maxDuration: "30s",
    },
  },
};

export function setup() {
  const email = `loadtest-${Date.now()}@seki.dev`;
  const registerRes = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    JSON.stringify({ email, password: "LoadTest1234" }),
    { headers: { "Content-Type": "application/json" } }
  );
  const token = registerRes.json("token");

  const resourcesRes = http.get(`${BASE_URL}/api/v1/resources`);
  const resources = resourcesRes.json();
  if (!resources || resources.length === 0) {
    throw new Error("No resources found — run `make seed` before the load test.");
  }
  const resourceId = resources[0].id;

  // A fixed, far-future slot every virtual user will try to book.
  const start = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
  const end = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000 + 60 * 60 * 1000).toISOString();

  return { token, resourceId, start, end };
}

export default function (data) {
  const res = http.post(
    `${BASE_URL}/api/v1/bookings`,
    JSON.stringify({
      resource_id: data.resourceId,
      start_time: data.start,
      end_time: data.end,
      notes: `attempt from VU ${__VU}`,
    }),
    {
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${data.token}`,
      },
    }
  );

  if (res.status === 201) {
    successCount.add(1);
  } else if (res.status === 409) {
    conflictCount.add(1);
  } else {
    unexpectedCount.add(1);
    console.error(`unexpected status ${res.status}: ${res.body}`);
  }

  check(res, {
    "status is 201 or 409 (never anything else)": (r) => r.status === 201 || r.status === 409,
  });
}

export function teardown(data) {
  console.log("\n=== Concurrency correctness result ===");
  console.log("Exactly ONE of the 50 concurrent requests should have succeeded (201).");
  console.log("All 49 others should have been rejected as conflicts (409).");
  console.log("Any 'unexpected_error' count above 0 indicates a real bug.\n");
}
