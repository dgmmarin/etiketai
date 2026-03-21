import { api } from "./client";

export interface CheckoutResponse {
  checkout_url: string;
}

export const billingApi = {
  createCheckout: (plan: string) =>
    api.post<CheckoutResponse>("/v1/billing/create-checkout", { plan }),
};
