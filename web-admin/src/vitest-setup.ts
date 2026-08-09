// Matchers reading the DOM the way a user would: toBeVisible, toBeDisabled,
// toHaveAccessibleName and friends.
import "@testing-library/jest-dom/vitest";

// Unmount whatever a test rendered, so the next one starts from a blank page.
import { cleanup } from "@testing-library/svelte";
import { afterEach } from "vitest";

afterEach(cleanup);
