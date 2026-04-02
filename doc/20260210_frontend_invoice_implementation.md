# Frontend Implementation Guide: Extended Invoice & Receipt Management

This document outlines the changes required in the frontend to support the extended receipt/invoice functionality implemented in the backend.

## 1. Data Models (TypeScript Interfaces)

Update your existing `Receipt` interface and create a new `InvoiceItem` interface.

```typescript
// Interface for a single line item on the invoice
export interface InvoiceItem {
  description: string;
  quantity: number;
  net_amount: number;      // Price per unit or total net for this line? Usually price * quantity
  tax_rate: number;        // e.g., 19.0 or 7.0
  vat_amount: number;      // Calculated: net_amount * (tax_rate / 100)
  gross_amount: number;    // Calculated: net_amount + vat_amount
}

// Updated Receipt Interface
export interface Receipt {
  id: string;
  club_id: string;
  // Existing fields
  type: 'income' | 'expense';
  recipient: string;       // Can be mapped to Buyer Name for display if needed
  number: string;          // Invoice Number (e.g., RE-2026-001)
  date: string;           // YYYY-MM-DD (Issue Date)
  position_assignment: string;
  amount: number;          // Total Gross Amount
  is_booked: boolean;
  note?: string;
  position_tax_account?: string;
  position_percentage?: number;
  donor_id?: string;
  
  // NEW FIELDS
  seller_name?: string;
  seller_address?: string;
  seller_tax_id?: string;
  seller_vat_id?: string;
  
  buyer_name?: string;
  buyer_address?: string; // Address field, could be multiline
  
  delivery_date?: string; // YYYY-MM-DD
  total_vat_amount: number; 
  invoice_items: InvoiceItem[]; // Array of items
  
  created_at: string;
  updated_at: string;
}

// Payload for Create/Update
export interface CreateUpdateReceiptPayload {
  type: string;
  recipient: string;
  number: string;
  date: string;
  position_assignment: string;
  amount: number;
  is_booked: boolean;
  note?: string;
  position_tax_account?: string;
  position_percentage?: string; // Backend expects string/number logic
  donor_id?: string;
  
  // New fields
  seller_name?: string;
  seller_address?: string;
  buyer_name?: string;
  buyer_address?: string;
  seller_tax_id?: string;
  seller_vat_id?: string;
  delivery_date?: string;
  total_vat_amount: number;
  invoice_items: InvoiceItem[];
}
```

## 2. API Service Updates

Extend your HTTP client service to handle the new CRUD operations.

```typescript
// Assuming axios or fetch wrapper
const BASE_URL = '/api/v1/finance/receipts';

export const ReceiptService = {
  getAll: () => http.get<Receipt[]>(BASE_URL),
  
  create: (data: CreateUpdateReceiptPayload) => http.post<Receipt>(BASE_URL, data),
  
  // NEW: Update existing receipt
  update: (id: string, data: CreateUpdateReceiptPayload) => http.put<Receipt>(`${BASE_URL}/${id}`, data),
  
  // NEW: Delete receipt
  delete: (id: string) => http.delete(`${BASE_URL}/${id}`),
};
```

## 3. UI Components & Logic

### A. Receipt List View
*   **Actions**: Add "Edit" (Pencil icon) and "Delete" (Trash icon) buttons to each row in the data grid.
*   **Columns**: You might want to display `Buyer Name` alongside `Recipient` (or merge them logically in standard view). The backend `recipient` field is still required, so you might want to auto-fill it with `buyer_name` on save.

### B. Receipt Form (Create / Edit)
The form needs to be significantly expanded. Consider using tabs or collapsible sections.

#### Section 1: General Info (Header)
*   **Invoice Number**: Text input.
*   **Type**: Dropdown (Income/Expense).
*   **Dates**:
    *   **Invoice Date**: Date picker (required).
    *   **Delivery/Service Date**: Date picker (optional).
*   **Assignment**: Dropdown (Ideel, Zweckbetrieb, etc.).

#### Section 2: Parties
*   **Seller (Us/Club)**:
    *   Inputs: Name, Address, Tax ID, VAT ID.
    *   *Feature*: Add a "Load Club Data" button to auto-fill this from the Club Settings (if available in frontend state).
*   **Buyer (Them)**:
    *   Inputs: Name, Address.
    *   *Synchronization*: When `Buyer Name` changes, update the legacy `Recipient` field automatically so sorting/filtering continues to work.

#### Section 3: Invoice Items (Dynamic Table)
This is the complex part. You need a dynamic list of items.

*   **Table Columns**:
    1.  **Description**: Text input.
    2.  **Quantity**: Number input.
    3.  **Net Price (Unit)**: Number input.
    4.  **Tax Rate (%)**: Dropdown (e.g., 0%, 7%, 19%) or Number input.
    5.  **VAT Amount**: Read-only (Calculated).
    6.  **Gross Amount**: Read-only (Calculated).
    7.  **Action**: "Remove" button.

*   **Calculation Logic (Reactive)**:
    *   `Row VAT` = `Quantity` * `Net Price` * (`Tax Rate` / 100)
    *   `Row Gross` = (`Quantity` * `Net Price`) + `Row VAT`
    
*   **Footer/Totals**:
    *   **Total Net**: Sum of all rows.
    *   **Total VAT**: Sum of all rows (maps to payload `total_vat_amount`).
    *   **Total Gross**: Sum of all rows (maps to payload `amount`).

#### Section 4: Settings/Meta
*   **Is Booked**: Checkbox.
*   **Notes**: Textarea.
*   **Link to Donor**: Lookup/Dropdown (optional).

### C. Validation Rules
*   **Required**: Number, Date, Type, Assignment.
*   **Logic**:
    *   Ensure at least one line item if `amount` is calculated from items.
    *   Or, allow manual override of Totals if the user doesn't want to type in items (make items optional in UI, but keep array empty in JSON).

## 4. Implementation Steps
1.  **Update API Client**: Add `update` and `delete` methods.
2.  **Refactor List**: Add Edit/Delete buttons. Check permissions (`finance:write`, `finance:delete`).
3.  **Build/Update Form**:
    *   Implement the "Invoice Items" array field logic (React `useFieldArray` or similar).
    *   Implement the live calculation of totals.
    *   Bind the new Seller/Buyer fields.
4.  **Test**: Verified creating a receipt with items -> checking list -> editing it -> checking values persistence.
