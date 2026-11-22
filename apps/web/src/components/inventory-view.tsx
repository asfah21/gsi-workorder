"use client";

import {
  ArrowDownToLine,
  ArrowLeftRight,
  ArrowUpFromLine,
  Boxes,
  CheckCircle2,
  CircleAlert,
  ClipboardCheck,
  ClipboardList,
  Loader2,
  Plus,
  RefreshCw,
  RotateCcw,
  Scale,
  Search,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
  Warehouse,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import { ActiveBadge } from "@/components/status-badges";
import { TablePagination } from "@/components/table-pagination";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { ApiError, proxyGet, proxyPatch, proxyPost } from "@/lib/api-client";
import type {
  DataResponse,
  InventoryBalance,
  InventoryMovement,
  InventoryReconciliation,
  InventoryTransaction,
  InventoryTransactionType,
  ListResponse,
  OpnameSession,
  Sparepart,
  StockPolicy,
  Warehouse as WarehouseType,
  WarehouseLocation,
} from "@/lib/types";
import { formatWita } from "@/lib/time";
import { cn } from "@/lib/utils";

type Tab = "stock" | "transactions" | "movements" | "opname" | "master";
type TxDraftItem = {
  key: string;
  sparepart_id: string;
  location_id: string;
  from_location_id: string;
  to_location_id: string;
  direction: "increase" | "decrease";
  qty: string;
  notes: string;
};
type TxForm = {
  reference_no: string;
  notes: string;
  idempotency_key: string;
  items: TxDraftItem[];
};

const txTypeLabels: Record<InventoryTransactionType, string> = {
  receipt: "Penerimaan",
  issue: "Pengeluaran",
  return: "Pengembalian",
  transfer: "Transfer",
  adjustment: "Adjustment",
  opname: "Stock Opname",
};
const txEndpoints: Record<Exclude<InventoryTransactionType, "opname">, string> = {
  receipt: "/inventory/receipts",
  issue: "/inventory/issues",
  return: "/inventory/returns",
  transfer: "/inventory/transfers",
  adjustment: "/inventory/adjustments",
};
const stockStatusLabels: Record<string, string> = {
  no_policy: "Tanpa policy",
  ok: "Aman",
  low: "Menipis",
  critical: "Kritis",
};
const opnameStatusLabels: Record<string, string> = {
  draft: "Draft",
  counting: "Counting",
  review: "Review",
  posted: "Posted",
  cancelled: "Dibatalkan",
};
const locationTypeLabels: Record<string, string> = {
  rack: "Rack",
  bin: "Bin",
  yard: "Yard",
  staging: "Staging",
};

const tabs: { id: Tab; label: string; icon: typeof Boxes }[] = [
  { id: "stock", label: "Stok", icon: Boxes },
  { id: "transactions", label: "Transaksi", icon: ArrowLeftRight },
  { id: "movements", label: "Pergerakan", icon: SlidersHorizontal },
  { id: "opname", label: "Stock Opname", icon: ClipboardList },
  { id: "master", label: "Master", icon: Warehouse },
];

function makeID() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `inv-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function emptyTxItem(): TxDraftItem {
  return {
    key: makeID(),
    sparepart_id: "",
    location_id: "",
    from_location_id: "",
    to_location_id: "",
    direction: "increase",
    qty: "1",
    notes: "",
  };
}

export function InventoryView() {
  const { user } = useSession();
  const isAdmin = user.role === "admin";
  const canMove = isAdmin || user.role === "storekeeper";
  const canApprove = isAdmin || user.role === "planner";
  const canCount = canMove;
  const canMaster = canApprove;

  const [tab, setTab] = useState<Tab>("stock");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const [spareparts, setSpareparts] = useState<Sparepart[]>([]);
  const [warehouses, setWarehouses] = useState<WarehouseType[]>([]);
  const [locations, setLocations] = useState<WarehouseLocation[]>([]);
  const [baseLoading, setBaseLoading] = useState(true);

  const [summary, setSummary] = useState({ low: 0, critical: 0 });

  const [balanceQueryInput, setBalanceQueryInput] = useState("");
  const [balanceQuery, setBalanceQuery] = useState("");
  const [balanceWarehouse, setBalanceWarehouse] = useState("");
  const [balanceStatus, setBalanceStatus] = useState("");
  const [balances, setBalances] = useState<InventoryBalance[]>([]);
  const [balanceTotal, setBalanceTotal] = useState(0);
  const [balancePage, setBalancePage] = useState(1);
  const [balancePageSize, setBalancePageSize] = useState(25);
  const [balanceLoading, setBalanceLoading] = useState(false);

  const [txTypeFilter, setTxTypeFilter] = useState<InventoryTransactionType>("receipt");
  const [txQuery, setTxQuery] = useState("");
  const [txFrom, setTxFrom] = useState("");
  const [txTo, setTxTo] = useState("");
  const [transactions, setTransactions] = useState<InventoryTransaction[]>([]);
  const [transactionTotal, setTransactionTotal] = useState(0);
  const [transactionPage, setTransactionPage] = useState(1);
  const [transactionPageSize, setTransactionPageSize] = useState(25);
  const [transactionLoading, setTransactionLoading] = useState(false);

  const [movementQuery, setMovementQuery] = useState("");
  const [movementType, setMovementType] = useState("");
  const [movementFrom, setMovementFrom] = useState("");
  const [movementTo, setMovementTo] = useState("");
  const [movements, setMovements] = useState<InventoryMovement[]>([]);
  const [movementTotal, setMovementTotal] = useState(0);
  const [movementPage, setMovementPage] = useState(1);
  const [movementPageSize, setMovementPageSize] = useState(25);
  const [movementLoading, setMovementLoading] = useState(false);

  const [opnameStatus, setOpnameStatus] = useState("");
  const [opnameWarehouse, setOpnameWarehouse] = useState("");
  const [opnameSessions, setOpnameSessions] = useState<OpnameSession[]>([]);
  const [opnameTotal, setOpnameTotal] = useState(0);
  const [opnamePage, setOpnamePage] = useState(1);
  const [opnamePageSize, setOpnamePageSize] = useState(25);
  const [opnameLoading, setOpnameLoading] = useState(false);

  const [policies, setPolicies] = useState<StockPolicy[]>([]);
  const [policyTotal, setPolicyTotal] = useState(0);
  const [policyPage, setPolicyPage] = useState(1);
  const [policyPageSize, setPolicyPageSize] = useState(25);
  const [policyLoading, setPolicyLoading] = useState(false);

  const [txFormOpen, setTxFormOpen] = useState(false);
  const [txFormType, setTxFormType] = useState<Exclude<InventoryTransactionType, "opname">>("receipt");
  const [txForm, setTxForm] = useState<TxForm>({
    reference_no: "",
    notes: "",
    idempotency_key: "",
    items: [emptyTxItem()],
  });
  const [txFormBusy, setTxFormBusy] = useState(false);
  const [txFormError, setTxFormError] = useState("");

  const [txDetailOpen, setTxDetailOpen] = useState(false);
  const [txDetail, setTxDetail] = useState<InventoryTransaction | null>(null);
  const [txDetailLoading, setTxDetailLoading] = useState(false);
  const [txDetailError, setTxDetailError] = useState("");
  const [reverseReason, setReverseReason] = useState("");
  const [reverseBusy, setReverseBusy] = useState(false);

  const [opnameCreateOpen, setOpnameCreateOpen] = useState(false);
  const [opnameCreateWarehouse, setOpnameCreateWarehouse] = useState("");
  const [opnameCreateNotes, setOpnameCreateNotes] = useState("");
  const [opnameCreateBusy, setOpnameCreateBusy] = useState(false);
  const [opnameCreateError, setOpnameCreateError] = useState("");

  const [opnameDetailOpen, setOpnameDetailOpen] = useState(false);
  const [opnameDetail, setOpnameDetail] = useState<OpnameSession | null>(null);
  const [opnameDetailLoading, setOpnameDetailLoading] = useState(false);
  const [opnameDetailError, setOpnameDetailError] = useState("");
  const [opnameCounts, setOpnameCounts] = useState<Record<number, string>>({});
  const [opnameCancelReason, setOpnameCancelReason] = useState("");
  const [opnameActionBusy, setOpnameActionBusy] = useState(false);

  const [warehouseForm, setWarehouseForm] = useState({ code: "", name: "", site_location: "" });
  const [locationForm, setLocationForm] = useState({
    warehouse_id: "",
    code: "",
    name: "",
    location_type: "rack",
  });
  const [policyEditingID, setPolicyEditingID] = useState<number | null>(null);
  const [policyForm, setPolicyForm] = useState({
    sparepart_id: "",
    warehouse_location_id: "",
    min_qty: "0",
    critical_qty: "0",
    reorder_qty: "",
    is_active: true,
  });
  const [masterBusy, setMasterBusy] = useState(false);
  const [masterError, setMasterError] = useState("");

  const [reconciliation, setReconciliation] = useState<InventoryReconciliation | null>(null);
  const [reconciliationLoading, setReconciliationLoading] = useState(false);

  const activeSpareparts = useMemo(() => spareparts.filter((item) => item.is_active), [spareparts]);
  const activeWarehouses = useMemo(() => warehouses.filter((item) => item.is_active), [warehouses]);
  const activeLocations = useMemo(() => locations.filter((item) => item.is_active), [locations]);

  const loadBase = useCallback(async () => {
    setBaseLoading(true);
    try {
      const [sparepartResult, warehouseResult, locationResult, lowResult, criticalResult] = await Promise.all([
        proxyGet<ListResponse<Sparepart>>("/spareparts?limit=100&active_only=true&sort=code&order=asc"),
        proxyGet<ListResponse<WarehouseType>>("/warehouses?limit=100&active_only=false"),
        proxyGet<DataResponse<WarehouseLocation[]>>("/warehouse-locations?active_only=false"),
        proxyGet<ListResponse<InventoryBalance>>("/inventory/balances?status=low&limit=1"),
        proxyGet<ListResponse<InventoryBalance>>("/inventory/balances?status=critical&limit=1"),
      ]);
      setSpareparts(sparepartResult.data);
      setWarehouses(warehouseResult.data);
      setLocations(locationResult.data);
      setSummary({ low: lowResult.meta.total, critical: criticalResult.meta.total });
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Data inventory tidak dapat dimuat.");
    } finally {
      setBaseLoading(false);
    }
  }, []);

  const loadBalances = useCallback(async () => {
    setBalanceLoading(true);
    try {
      const params = new URLSearchParams({ page: String(balancePage), limit: String(balancePageSize) });
      if (balanceQuery.trim()) params.set("q", balanceQuery.trim());
      if (balanceWarehouse) params.set("warehouse_id", balanceWarehouse);
      if (balanceStatus) params.set("status", balanceStatus);
      const result = await proxyGet<ListResponse<InventoryBalance>>(`/inventory/balances?${params.toString()}`);
      setBalances(result.data);
      setBalanceTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Saldo stok tidak dapat dimuat.");
    } finally {
      setBalanceLoading(false);
    }
  }, [balancePage, balancePageSize, balanceQuery, balanceStatus, balanceWarehouse]);

  const loadTransactions = useCallback(async () => {
    if (txTypeFilter === "opname") return;
    setTransactionLoading(true);
    try {
      const params = new URLSearchParams({ page: String(transactionPage), limit: String(transactionPageSize) });
      if (txQuery.trim()) params.set("q", txQuery.trim());
      if (txFrom) params.set("from", txFrom);
      if (txTo) params.set("to", txTo);
      const result = await proxyGet<ListResponse<InventoryTransaction>>(
        `${txEndpoints[txTypeFilter]}?${params.toString()}`,
      );
      setTransactions(result.data);
      setTransactionTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Transaksi inventory tidak dapat dimuat.");
    } finally {
      setTransactionLoading(false);
    }
  }, [transactionPage, transactionPageSize, txFrom, txQuery, txTo, txTypeFilter]);

  const loadMovements = useCallback(async () => {
    setMovementLoading(true);
    try {
      const params = new URLSearchParams({ page: String(movementPage), limit: String(movementPageSize) });
      if (movementQuery.trim()) params.set("q", movementQuery.trim());
      if (movementType) params.set("type", movementType);
      if (movementFrom) params.set("from", movementFrom);
      if (movementTo) params.set("to", movementTo);
      const result = await proxyGet<ListResponse<InventoryMovement>>(`/inventory/movements?${params.toString()}`);
      setMovements(result.data);
      setMovementTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Pergerakan stok tidak dapat dimuat.");
    } finally {
      setMovementLoading(false);
    }
  }, [movementFrom, movementPage, movementPageSize, movementQuery, movementTo, movementType]);

  const loadOpnames = useCallback(async () => {
    setOpnameLoading(true);
    try {
      const params = new URLSearchParams({ page: String(opnamePage), limit: String(opnamePageSize) });
      if (opnameStatus) params.set("status", opnameStatus);
      if (opnameWarehouse) params.set("warehouse_id", opnameWarehouse);
      const result = await proxyGet<ListResponse<OpnameSession>>(
        `/inventory/opname/sessions?${params.toString()}`,
      );
      setOpnameSessions(result.data);
      setOpnameTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Sesi stock opname tidak dapat dimuat.");
    } finally {
      setOpnameLoading(false);
    }
  }, [opnamePage, opnamePageSize, opnameStatus, opnameWarehouse]);

  const loadPolicies = useCallback(async () => {
    setPolicyLoading(true);
    try {
      const params = new URLSearchParams({ page: String(policyPage), limit: String(policyPageSize) });
      const result = await proxyGet<ListResponse<StockPolicy>>(`/inventory/stock-policies?${params.toString()}`);
      setPolicies(result.data);
      setPolicyTotal(result.meta.total);
    } catch (loadError) {
      setError(loadError instanceof ApiError ? loadError.message : "Stock policy tidak dapat dimuat.");
    } finally {
      setPolicyLoading(false);
    }
  }, [policyPage, policyPageSize]);

  useEffect(() => {
    void loadBase();
  }, [loadBase]);

  useEffect(() => {
    if (tab === "stock") void loadBalances();
  }, [loadBalances, tab]);

  useEffect(() => {
    if (tab === "transactions") void loadTransactions();
  }, [loadTransactions, tab]);

  useEffect(() => {
    if (tab === "movements") void loadMovements();
  }, [loadMovements, tab]);

  useEffect(() => {
    if (tab === "opname") void loadOpnames();
  }, [loadOpnames, tab]);

  useEffect(() => {
    if (tab === "master") {
      void loadPolicies();
      void loadBase();
    }
  }, [loadBase, loadPolicies, tab]);

  const refreshAfterWrite = useCallback(async () => {
    await loadBase();
    await Promise.all([
      loadBalances(),
      loadTransactions(),
      loadMovements(),
      loadOpnames(),
      loadPolicies(),
    ]);
  }, [loadBalances, loadBase, loadMovements, loadOpnames, loadPolicies, loadTransactions]);

  const canPostType = (type: Exclude<InventoryTransactionType, "opname">) => {
    return type === "adjustment" ? canApprove : canMove;
  };

  const openTxForm = (type: Exclude<InventoryTransactionType, "opname">) => {
    setTxFormType(type);
    setTxForm({
      reference_no: "",
      notes: "",
      idempotency_key: makeID(),
      items: [emptyTxItem()],
    });
    setTxFormError("");
    setTxFormOpen(true);
  };

  const updateTxItem = (key: string, patch: Partial<TxDraftItem>) => {
    setTxForm((current) => ({
      ...current,
      items: current.items.map((item) => (item.key === key ? { ...item, ...patch } : item)),
    }));
  };

  const removeTxItem = (key: string) => {
    setTxForm((current) => ({
      ...current,
      items: current.items.length === 1 ? current.items : current.items.filter((item) => item.key !== key),
    }));
  };

  const submitTx = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (txFormBusy) return;
    const notes = txForm.notes.trim();
    if ((txFormType === "issue" || txFormType === "adjustment") && !notes) {
      setTxFormError("Catatan/alasan wajib diisi untuk transaksi ini.");
      return;
    }
    const mapped: Record<string, unknown>[] = [];
    for (const item of txForm.items) {
      const sparepartID = Number(item.sparepart_id);
      const qty = Number(item.qty);
      if (!sparepartID) {
        setTxFormError("Pilih sparepart untuk setiap baris.");
        return;
      }
      if (!Number.isFinite(qty) || qty <= 0) {
        setTxFormError("Qty setiap baris harus lebih dari 0.");
        return;
      }
      const base = { sparepart_id: sparepartID, qty, notes: item.notes.trim() };
      if (txFormType === "receipt" || txFormType === "return") {
        const locationID = Number(item.location_id);
        if (!locationID) {
          setTxFormError("Lokasi tujuan wajib dipilih.");
          return;
        }
        mapped.push({ ...base, to_location_id: locationID });
      } else if (txFormType === "issue") {
        const locationID = Number(item.location_id);
        if (!locationID) {
          setTxFormError("Lokasi asal wajib dipilih.");
          return;
        }
        mapped.push({ ...base, from_location_id: locationID });
      } else if (txFormType === "transfer") {
        const fromID = Number(item.from_location_id);
        const toID = Number(item.to_location_id);
        if (!fromID || !toID) {
          setTxFormError("Lokasi asal dan tujuan transfer wajib dipilih.");
          return;
        }
        if (fromID === toID) {
          setTxFormError("Lokasi asal dan tujuan transfer tidak boleh sama.");
          return;
        }
        mapped.push({ ...base, from_location_id: fromID, to_location_id: toID });
      } else {
        const locationID = Number(item.location_id);
        if (!locationID) {
          setTxFormError("Lokasi adjustment wajib dipilih.");
          return;
        }
        if (item.direction === "increase") {
          mapped.push({ ...base, to_location_id: locationID });
        } else {
          mapped.push({ ...base, from_location_id: locationID });
        }
      }
    }

    setTxFormBusy(true);
    setTxFormError("");
    try {
      await proxyPost<DataResponse<InventoryTransaction>>(txEndpoints[txFormType], {
        reference_type: "manual",
        reference_no: txForm.reference_no.trim(),
        notes,
        idempotency_key: txForm.idempotency_key.trim() || makeID(),
        items: mapped,
      });
      setSuccess(`${txTypeLabels[txFormType]} berhasil diposting.`);
      setTxFormOpen(false);
      setTab("transactions");
      setTxTypeFilter(txFormType);
      setTransactionPage(1);
      await refreshAfterWrite();
    } catch (saveError) {
      setTxFormError(saveError instanceof ApiError ? saveError.message : "Transaksi tidak dapat diposting.");
    } finally {
      setTxFormBusy(false);
    }
  };

  const openTransaction = async (id: number) => {
    setTxDetailOpen(true);
    setTxDetail(null);
    setTxDetailError("");
    setReverseReason("");
    setTxDetailLoading(true);
    try {
      const result = await proxyGet<DataResponse<InventoryTransaction>>(`/inventory/transactions/${id}`);
      setTxDetail(result.data);
    } catch (loadError) {
      setTxDetailError(loadError instanceof ApiError ? loadError.message : "Detail transaksi tidak dapat dimuat.");
    } finally {
      setTxDetailLoading(false);
    }
  };

  const submitReverse = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!txDetail || reverseBusy) return;
    if (!reverseReason.trim()) {
      setTxDetailError("Alasan reversal wajib diisi.");
      return;
    }
    setReverseBusy(true);
    setTxDetailError("");
    try {
      await proxyPost<DataResponse<InventoryTransaction>>(
        `/inventory/transactions/${txDetail.id}/reverse`,
        { reason: reverseReason.trim() },
      );
      setSuccess(`Reversal untuk ${txDetail.code} berhasil diposting.`);
      setTxDetailOpen(false);
      await refreshAfterWrite();
    } catch (saveError) {
      setTxDetailError(saveError instanceof ApiError ? saveError.message : "Reversal tidak dapat diposting.");
    } finally {
      setReverseBusy(false);
    }
  };

  const openOpnameCreate = () => {
    setOpnameCreateWarehouse(activeWarehouses[0] ? String(activeWarehouses[0].id) : "");
    setOpnameCreateNotes("");
    setOpnameCreateError("");
    setOpnameCreateOpen(true);
  };

  const submitOpnameCreate = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (opnameCreateBusy) return;
    if (!opnameCreateWarehouse) {
      setOpnameCreateError("Pilih gudang terlebih dahulu.");
      return;
    }
    setOpnameCreateBusy(true);
    setOpnameCreateError("");
    try {
      const result = await proxyPost<DataResponse<OpnameSession>>("/inventory/opname/sessions", {
        warehouse_id: Number(opnameCreateWarehouse),
        notes: opnameCreateNotes.trim(),
      });
      setSuccess(`Sesi opname ${result.data.code} dibuat.`);
      setOpnameCreateOpen(false);
      setOpnameDetail(result.data);
      setOpnameCounts(
        Object.fromEntries(result.data.items.map((item) => [item.id, item.counted_qty?.toString() ?? ""])),
      );
      setOpnameDetailOpen(true);
      await refreshAfterWrite();
    } catch (saveError) {
      setOpnameCreateError(saveError instanceof ApiError ? saveError.message : "Sesi opname tidak dapat dibuat.");
    } finally {
      setOpnameCreateBusy(false);
    }
  };

  const openOpname = async (id: number) => {
    setOpnameDetailOpen(true);
    setOpnameDetail(null);
    setOpnameDetailError("");
    setOpnameCancelReason("");
    setOpnameDetailLoading(true);
    try {
      const result = await proxyGet<DataResponse<OpnameSession>>(`/inventory/opname/sessions/${id}`);
      setOpnameDetail(result.data);
      setOpnameCounts(
        Object.fromEntries(result.data.items.map((item) => [item.id, item.counted_qty?.toString() ?? ""])),
      );
    } catch (loadError) {
      setOpnameDetailError(loadError instanceof ApiError ? loadError.message : "Sesi opname tidak dapat dimuat.");
    } finally {
      setOpnameDetailLoading(false);
    }
  };

  const saveCounts = async () => {
    if (!opnameDetail || opnameActionBusy) return;
    const payload = opnameDetail.items
      .filter((item) => (opnameCounts[item.id] ?? "").trim() !== "")
      .map((item) => ({
        item_id: item.id,
        counted_qty: Number(opnameCounts[item.id]),
        notes: item.notes.trim(),
      }));
    if (payload.length === 0) {
      setOpnameDetailError("Isi minimal satu qty hitung.");
      return;
    }
    if (payload.some((item) => !Number.isFinite(item.counted_qty) || item.counted_qty < 0)) {
      setOpnameDetailError("Qty hitung tidak boleh negatif.");
      return;
    }
    setOpnameActionBusy(true);
    setOpnameDetailError("");
    try {
      const result = await proxyPatch<DataResponse<OpnameSession>>(
        `/inventory/opname/sessions/${opnameDetail.id}/items`,
        { items: payload },
      );
      setOpnameDetail(result.data);
      setOpnameCounts(
        Object.fromEntries(result.data.items.map((item) => [item.id, item.counted_qty?.toString() ?? ""])),
      );
      setSuccess("Hasil hitung opname disimpan.");
      await refreshAfterWrite();
    } catch (saveError) {
      setOpnameDetailError(saveError instanceof ApiError ? saveError.message : "Hasil hitung tidak dapat disimpan.");
    } finally {
      setOpnameActionBusy(false);
    }
  };

  const postOpname = async () => {
    if (!opnameDetail || opnameActionBusy) return;
    if (!window.confirm("Posting opname akan membuat adjustment permanen untuk setiap selisih. Lanjutkan?")) return;
    setOpnameActionBusy(true);
    setOpnameDetailError("");
    try {
      const result = await proxyPost<DataResponse<OpnameSession>>(
        `/inventory/opname/sessions/${opnameDetail.id}/post`,
        {},
      );
      setOpnameDetail(result.data);
      setSuccess(`Sesi opname ${result.data.code} diposting.`);
      await refreshAfterWrite();
    } catch (saveError) {
      setOpnameDetailError(saveError instanceof ApiError ? saveError.message : "Sesi opname tidak dapat diposting.");
    } finally {
      setOpnameActionBusy(false);
    }
  };

  const cancelOpname = async () => {
    if (!opnameDetail || opnameActionBusy) return;
    if (!opnameCancelReason.trim()) {
      setOpnameDetailError("Alasan pembatalan wajib diisi.");
      return;
    }
    setOpnameActionBusy(true);
    setOpnameDetailError("");
    try {
      const result = await proxyPost<DataResponse<OpnameSession>>(
        `/inventory/opname/sessions/${opnameDetail.id}/cancel`,
        { reason: opnameCancelReason.trim() },
      );
      setOpnameDetail(result.data);
      setSuccess(`Sesi opname ${result.data.code} dibatalkan.`);
      await refreshAfterWrite();
    } catch (saveError) {
      setOpnameDetailError(saveError instanceof ApiError ? saveError.message : "Sesi opname tidak dapat dibatalkan.");
    } finally {
      setOpnameActionBusy(false);
    }
  };

  const submitWarehouse = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (masterBusy) return;
    setMasterBusy(true);
    setMasterError("");
    try {
      const result = await proxyPost<DataResponse<WarehouseType>>("/warehouses", {
        code: warehouseForm.code.trim(),
        name: warehouseForm.name.trim(),
        site_location: warehouseForm.site_location.trim(),
      });
      setSuccess(`Gudang ${result.data.code} dibuat.`);
      setWarehouseForm({ code: "", name: "", site_location: "" });
      await refreshAfterWrite();
    } catch (saveError) {
      setMasterError(saveError instanceof ApiError ? saveError.message : "Gudang tidak dapat disimpan.");
    } finally {
      setMasterBusy(false);
    }
  };

  const toggleWarehouse = async (warehouse: WarehouseType) => {
    setMasterError("");
    try {
      await proxyPatch<DataResponse<WarehouseType>>(`/warehouses/${warehouse.id}`, {
        is_active: !warehouse.is_active,
      });
      setSuccess(`Status gudang ${warehouse.code} diperbarui.`);
      await refreshAfterWrite();
    } catch (saveError) {
      setMasterError(saveError instanceof ApiError ? saveError.message : "Status gudang tidak dapat diubah.");
    }
  };

  const submitLocation = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (masterBusy) return;
    if (!locationForm.warehouse_id) {
      setMasterError("Pilih gudang untuk lokasi baru.");
      return;
    }
    setMasterBusy(true);
    setMasterError("");
    try {
      const result = await proxyPost<DataResponse<WarehouseLocation>>(
        `/warehouses/${locationForm.warehouse_id}/locations`,
        {
          code: locationForm.code.trim(),
          name: locationForm.name.trim(),
          location_type: locationForm.location_type,
        },
      );
      setSuccess(`Lokasi ${result.data.code} dibuat.`);
      setLocationForm((current) => ({ ...current, code: "", name: "" }));
      await refreshAfterWrite();
    } catch (saveError) {
      setMasterError(saveError instanceof ApiError ? saveError.message : "Lokasi tidak dapat disimpan.");
    } finally {
      setMasterBusy(false);
    }
  };

  const toggleLocation = async (location: WarehouseLocation) => {
    setMasterError("");
    try {
      await proxyPatch<DataResponse<WarehouseLocation>>(`/warehouse-locations/${location.id}`, {
        is_active: !location.is_active,
      });
      setSuccess(`Status lokasi ${location.code} diperbarui.`);
      await refreshAfterWrite();
    } catch (saveError) {
      setMasterError(saveError instanceof ApiError ? saveError.message : "Status lokasi tidak dapat diubah.");
    }
  };

  const resetPolicyForm = () => {
    setPolicyEditingID(null);
    setPolicyForm({
      sparepart_id: "",
      warehouse_location_id: "",
      min_qty: "0",
      critical_qty: "0",
      reorder_qty: "",
      is_active: true,
    });
  };

  const editPolicy = (policy: StockPolicy) => {
    setPolicyEditingID(policy.id);
    setPolicyForm({
      sparepart_id: String(policy.sparepart_id),
      warehouse_location_id: String(policy.warehouse_location_id),
      min_qty: String(policy.min_qty),
      critical_qty: String(policy.critical_qty),
      reorder_qty: policy.reorder_qty === undefined ? "" : String(policy.reorder_qty),
      is_active: policy.is_active,
    });
    setMasterError("");
  };

  const submitPolicy = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (masterBusy) return;
    if (!policyForm.sparepart_id || !policyForm.warehouse_location_id) {
      setMasterError("Sparepart dan lokasi policy wajib dipilih.");
      return;
    }
    const minQty = Number(policyForm.min_qty);
    const criticalQty = Number(policyForm.critical_qty);
    const reorderQty = policyForm.reorder_qty.trim() === "" ? null : Number(policyForm.reorder_qty);
    if (!Number.isFinite(minQty) || !Number.isFinite(criticalQty) || minQty < 0 || criticalQty < 0) {
      setMasterError("Min dan critical qty tidak boleh negatif.");
      return;
    }
    if (reorderQty !== null && (!Number.isFinite(reorderQty) || reorderQty < 0)) {
      setMasterError("Reorder qty tidak boleh negatif.");
      return;
    }
    setMasterBusy(true);
    setMasterError("");
    try {
      if (policyEditingID) {
        await proxyPatch<DataResponse<StockPolicy>>(`/inventory/stock-policies/${policyEditingID}`, {
          min_qty: minQty,
          critical_qty: criticalQty,
          reorder_qty: reorderQty ?? undefined,
          clear_reorder_qty: reorderQty === null,
          is_active: policyForm.is_active,
        });
        setSuccess("Stock policy diperbarui.");
      } else {
        await proxyPost<DataResponse<StockPolicy>>("/inventory/stock-policies", {
          sparepart_id: Number(policyForm.sparepart_id),
          warehouse_location_id: Number(policyForm.warehouse_location_id),
          min_qty: minQty,
          critical_qty: criticalQty,
          reorder_qty: reorderQty ?? undefined,
          is_active: policyForm.is_active,
        });
        setSuccess("Stock policy disimpan.");
      }
      resetPolicyForm();
      await refreshAfterWrite();
    } catch (saveError) {
      setMasterError(saveError instanceof ApiError ? saveError.message : "Stock policy tidak dapat disimpan.");
    } finally {
      setMasterBusy(false);
    }
  };

  const checkReconciliation = async () => {
    setReconciliationLoading(true);
    setMasterError("");
    try {
      const result = await proxyGet<DataResponse<InventoryReconciliation>>("/inventory/reconciliation");
      setReconciliation(result.data);
      setSuccess(result.data.mismatch === 0 ? "Ledger dan saldo seimbang." : `${result.data.mismatch} selisih ditemukan.`);
    } catch (loadError) {
      setMasterError(loadError instanceof ApiError ? loadError.message : "Rekonsiliasi tidak dapat dijalankan.");
    } finally {
      setReconciliationLoading(false);
    }
  };

  const balancePages = Math.max(1, Math.ceil(balanceTotal / balancePageSize));
  const transactionPages = Math.max(1, Math.ceil(transactionTotal / transactionPageSize));
  const movementPages = Math.max(1, Math.ceil(movementTotal / movementPageSize));
  const opnamePages = Math.max(1, Math.ceil(opnameTotal / opnamePageSize));
  const policyPages = Math.max(1, Math.ceil(policyTotal / policyPageSize));
  const opnameHasUnsavedCounts = opnameDetail
    ? opnameDetail.items.some(
        (item) => (opnameCounts[item.id] ?? "").trim() !== (item.counted_qty?.toString() ?? ""),
      )
    : false;

  const locationLabel = (location: WarehouseLocation) =>
    `${location.warehouse_code} / ${location.code} - ${location.name}`;

  const sparepartLabel = (sparepart: Sparepart) => `${sparepart.code} - ${sparepart.name}`;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Inventory</h1>
          <p className="text-sm text-muted-foreground">
            Stok sparepart, transaksi immutable, stock opname, dan minimum stock per lokasi.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {tab === "transactions" && canMove ? (
            <>
              <Button type="button" size="sm" onClick={() => openTxForm("receipt")}>
                <ArrowDownToLine aria-hidden="true" /> Penerimaan
              </Button>
              <Button type="button" size="sm" variant="outline" onClick={() => openTxForm("issue")}>
                <ArrowUpFromLine aria-hidden="true" /> Pengeluaran
              </Button>
            </>
          ) : null}
          {tab === "opname" && canCount ? (
            <Button type="button" onClick={openOpnameCreate}>
              <Plus aria-hidden="true" /> Sesi opname
            </Button>
          ) : null}
          <Button type="button" variant="outline" onClick={() => void refreshAfterWrite()}>
            <RefreshCw aria-hidden="true" /> Muat ulang
          </Button>
        </div>
      </div>

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      {success ? (
        <Alert className="border-emerald-500/30 bg-emerald-500/5">
          <CheckCircle2 aria-hidden="true" />
          <AlertTitle>Berhasil</AlertTitle>
          <AlertDescription>{success}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Card className="gap-2 py-4">
          <CardHeader className="flex flex-row items-center justify-between px-4">
            <CardTitle className="text-sm font-medium">Stok menipis</CardTitle>
            <ArrowDownToLine className="size-4 text-amber-600" aria-hidden="true" />
          </CardHeader>
          <CardContent className="px-4">
            <p className="text-2xl font-semibold">{baseLoading ? "..." : summary.low}</p>
            <p className="text-xs text-muted-foreground">Lokasi berstatus low</p>
          </CardContent>
        </Card>
        <Card className="gap-2 py-4">
          <CardHeader className="flex flex-row items-center justify-between px-4">
            <CardTitle className="text-sm font-medium">Stok kritis</CardTitle>
            <CircleAlert className="size-4 text-destructive" aria-hidden="true" />
          </CardHeader>
          <CardContent className="px-4">
            <p className="text-2xl font-semibold">{baseLoading ? "..." : summary.critical}</p>
            <p className="text-xs text-muted-foreground">Lokasi berstatus critical</p>
          </CardContent>
        </Card>
        <Card className="gap-2 py-4">
          <CardHeader className="flex flex-row items-center justify-between px-4">
            <CardTitle className="text-sm font-medium">Gudang</CardTitle>
            <Warehouse className="size-4 text-muted-foreground" aria-hidden="true" />
          </CardHeader>
          <CardContent className="px-4">
            <p className="text-2xl font-semibold">{baseLoading ? "..." : warehouses.length}</p>
            <p className="text-xs text-muted-foreground">{activeWarehouses.length} aktif</p>
          </CardContent>
        </Card>
        <Card className="gap-2 py-4">
          <CardHeader className="flex flex-row items-center justify-between px-4">
            <CardTitle className="text-sm font-medium">Lokasi</CardTitle>
            <Boxes className="size-4 text-muted-foreground" aria-hidden="true" />
          </CardHeader>
          <CardContent className="px-4">
            <p className="text-2xl font-semibold">{baseLoading ? "..." : locations.length}</p>
            <p className="text-xs text-muted-foreground">{activeLocations.length} aktif</p>
          </CardContent>
        </Card>
      </div>

      <div className="flex gap-2 overflow-x-auto pb-1" role="tablist" aria-label="Bagian inventory">
        {tabs.map((item) => {
          const Icon = item.icon;
          return (
            <Button
              key={item.id}
              type="button"
              size="sm"
              role="tab"
              aria-selected={tab === item.id}
              variant={tab === item.id ? "default" : "outline"}
              onClick={() => setTab(item.id)}
              className="shrink-0"
            >
              <Icon aria-hidden="true" /> {item.label}
            </Button>
          );
        })}
      </div>

      {tab === "stock" ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Boxes className="size-4" aria-hidden="true" /> Saldo stok
            </CardTitle>
            <CardDescription>
              Saldo tersedia per sparepart dan lokasi. Reserved belum dipakai pada M5.
            </CardDescription>
            <form
              className="grid gap-3 md:grid-cols-2 xl:grid-cols-5"
              onSubmit={(event) => {
                event.preventDefault();
                setBalancePage(1);
                setBalanceQuery(balanceQueryInput);
              }}
            >
              <Field className="xl:col-span-2">
                <FieldLabel htmlFor="stock-search">Cari sparepart</FieldLabel>
                <div className="relative">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                  <Input
                    id="stock-search"
                    value={balanceQueryInput}
                    onChange={(event) => setBalanceQueryInput(event.target.value)}
                    placeholder="Kode atau nama sparepart..."
                    className="pl-9"
                  />
                </div>
              </Field>
              <Field>
                <FieldLabel htmlFor="stock-warehouse">Gudang</FieldLabel>
                <NativeSelect
                  id="stock-warehouse"
                  value={balanceWarehouse}
                  onChange={(event) => {
                    setBalanceWarehouse(event.target.value);
                    setBalancePage(1);
                  }}
                >
                  <NativeSelectOption value="">Semua gudang</NativeSelectOption>
                  {warehouses.map((warehouse) => (
                    <NativeSelectOption key={warehouse.id} value={warehouse.id}>
                      {warehouse.code} - {warehouse.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="stock-status">Status</FieldLabel>
                <NativeSelect
                  id="stock-status"
                  value={balanceStatus}
                  onChange={(event) => {
                    setBalanceStatus(event.target.value);
                    setBalancePage(1);
                  }}
                >
                  <NativeSelectOption value="">Semua status</NativeSelectOption>
                  {Object.entries(stockStatusLabels).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <div className="flex items-end gap-2">
                <Button type="submit"><Search aria-hidden="true" /> Cari</Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setBalanceQueryInput("");
                    setBalanceQuery("");
                    setBalanceWarehouse("");
                    setBalanceStatus("");
                    setBalancePage(1);
                  }}
                >
                  Reset
                </Button>
              </div>
            </form>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {balanceLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : balances.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon"><Boxes aria-hidden="true" /></EmptyMedia>
                  <EmptyTitle>Tidak ada saldo stok</EmptyTitle>
                  <EmptyDescription>Ubah filter atau posting penerimaan pertama.</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Sparepart</TableHead>
                      <TableHead>Lokasi</TableHead>
                      <TableHead>On hand</TableHead>
                      <TableHead>Reserved</TableHead>
                      <TableHead>Available</TableHead>
                      <TableHead>Min / Critical</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Update</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {balances.map((balance) => (
                      <TableRow key={`${balance.sparepart_id}-${balance.warehouse_location_id}`}>
                        <TableCell>
                          <p className="font-medium">{balance.sparepart_code}</p>
                          <p className="text-xs text-muted-foreground">{balance.sparepart_name}</p>
                        </TableCell>
                        <TableCell>
                          <p>{balance.warehouse_code} / {balance.location_code}</p>
                          <p className="text-xs text-muted-foreground">{balance.location_name}</p>
                        </TableCell>
                        <TableCell className="whitespace-nowrap">{balance.on_hand} {balance.uom}</TableCell>
                        <TableCell>{balance.reserved}</TableCell>
                        <TableCell className="font-medium whitespace-nowrap">{balance.available} {balance.uom}</TableCell>
                        <TableCell className="whitespace-nowrap text-sm">
                          {balance.min_qty} / {balance.critical_qty}
                        </TableCell>
                        <TableCell><StockStatusBadge status={balance.stock_status} /></TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {balance.updated_at ? formatWita(balance.updated_at) : "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            <TablePagination
              page={balancePage}
              pageSize={balancePageSize}
              total={balanceTotal}
              totalPages={balancePages}
              loading={balanceLoading}
              itemLabel="saldo"
              onPageChange={setBalancePage}
              onPageSizeChange={(size) => { setBalancePageSize(size); setBalancePage(1); }}
            />
          </CardContent>
        </Card>
      ) : null}

      {tab === "transactions" ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ArrowLeftRight className="size-4" aria-hidden="true" /> Transaksi inventory
            </CardTitle>
            <CardDescription>
              Transaksi posted tidak dapat diedit. Koreksi dilakukan dengan reversal.
            </CardDescription>
            <div className="flex flex-wrap gap-2">
              {(["receipt", "issue", "return", "transfer", "adjustment"] as const).map((type) =>
                canPostType(type) ? (
                  <Button key={type} type="button" size="sm" variant="outline" onClick={() => openTxForm(type)}>
                    {type === "receipt" ? <ArrowDownToLine aria-hidden="true" /> : null}
                    {type === "issue" ? <ArrowUpFromLine aria-hidden="true" /> : null}
                    {type === "transfer" ? <ArrowLeftRight aria-hidden="true" /> : null}
                    {type === "adjustment" ? <Scale aria-hidden="true" /> : null}
                    {type === "return" ? <RotateCcw aria-hidden="true" /> : null}
                    Catat {txTypeLabels[type]}
                  </Button>
                ) : null,
              )}
            </div>
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
              <Field>
                <FieldLabel htmlFor="tx-type">Jenis transaksi</FieldLabel>
                <NativeSelect
                  id="tx-type"
                  value={txTypeFilter}
                  onChange={(event) => {
                    setTxTypeFilter(event.target.value as InventoryTransactionType);
                    setTransactionPage(1);
                  }}
                >
                  {Object.entries(txEndpoints).map(([type]) => (
                    <NativeSelectOption key={type} value={type}>
                      {txTypeLabels[type as InventoryTransactionType]}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field className="xl:col-span-2">
                <FieldLabel htmlFor="tx-search">Cari kode/referensi</FieldLabel>
                <Input
                  id="tx-search"
                  value={txQuery}
                  onChange={(event) => { setTxQuery(event.target.value); setTransactionPage(1); }}
                  placeholder="Kode transaksi atau nomor referensi..."
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="tx-from">Dari tanggal</FieldLabel>
                <Input id="tx-from" type="date" value={txFrom} onChange={(event) => { setTxFrom(event.target.value); setTransactionPage(1); }} />
              </Field>
              <Field>
                <FieldLabel htmlFor="tx-to">Sampai tanggal</FieldLabel>
                <Input id="tx-to" type="date" value={txTo} onChange={(event) => { setTxTo(event.target.value); setTransactionPage(1); }} />
              </Field>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {transactionLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : transactions.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon"><ArrowLeftRight aria-hidden="true" /></EmptyMedia>
                  <EmptyTitle>Belum ada transaksi</EmptyTitle>
                  <EmptyDescription>Transaksi {txTypeLabels[txTypeFilter]} belum tercatat.</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Kode</TableHead>
                      <TableHead>Waktu</TableHead>
                      <TableHead>Referensi</TableHead>
                      <TableHead>Item</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Posted by</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {transactions.map((transaction) => (
                      <TableRow key={transaction.id}>
                        <TableCell>
                          <button
                            type="button"
                            className="cursor-pointer text-left font-medium hover:underline"
                            onClick={() => void openTransaction(transaction.id)}
                          >
                            {transaction.code}
                          </button>
                          {transaction.reversal_of_id ? (
                            <p className="text-xs text-muted-foreground">Reversal #{transaction.reversal_of_id}</p>
                          ) : null}
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-sm">{formatWita(transaction.posted_at)}</TableCell>
                        <TableCell className="max-w-56">
                          <p>{transaction.reference_no || "-"}</p>
                          <p className="truncate text-xs text-muted-foreground">{transaction.notes || "-"}</p>
                        </TableCell>
                        <TableCell>{transaction.item_count}</TableCell>
                        <TableCell><TransactionStatusBadge status={transaction.status} /></TableCell>
                        <TableCell className="text-sm">{transaction.posted_by_name || "-"}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            <TablePagination
              page={transactionPage}
              pageSize={transactionPageSize}
              total={transactionTotal}
              totalPages={transactionPages}
              loading={transactionLoading}
              itemLabel="transaksi"
              onPageChange={setTransactionPage}
              onPageSizeChange={(size) => { setTransactionPageSize(size); setTransactionPage(1); }}
            />
          </CardContent>
        </Card>
      ) : null}

      {tab === "movements" ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <SlidersHorizontal className="size-4" aria-hidden="true" /> Pergerakan stok
            </CardTitle>
            <CardDescription>Ledger immutable per item transaksi, termasuk transaksi reversal.</CardDescription>
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
              <Field className="xl:col-span-2">
                <FieldLabel htmlFor="movement-search">Cari</FieldLabel>
                <Input
                  id="movement-search"
                  value={movementQuery}
                  onChange={(event) => { setMovementQuery(event.target.value); setMovementPage(1); }}
                  placeholder="Kode transaksi, sparepart, referensi..."
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="movement-type">Jenis</FieldLabel>
                <NativeSelect
                  id="movement-type"
                  value={movementType}
                  onChange={(event) => { setMovementType(event.target.value); setMovementPage(1); }}
                >
                  <NativeSelectOption value="">Semua jenis</NativeSelectOption>
                  {Object.entries(txTypeLabels).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="movement-from">Dari tanggal</FieldLabel>
                <Input id="movement-from" type="date" value={movementFrom} onChange={(event) => { setMovementFrom(event.target.value); setMovementPage(1); }} />
              </Field>
              <Field>
                <FieldLabel htmlFor="movement-to">Sampai tanggal</FieldLabel>
                <Input id="movement-to" type="date" value={movementTo} onChange={(event) => { setMovementTo(event.target.value); setMovementPage(1); }} />
              </Field>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {movementLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : movements.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon"><SlidersHorizontal aria-hidden="true" /></EmptyMedia>
                  <EmptyTitle>Belum ada pergerakan</EmptyTitle>
                  <EmptyDescription>Ledger akan terisi setelah transaksi diposting.</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Waktu</TableHead>
                      <TableHead>Transaksi</TableHead>
                      <TableHead>Sparepart</TableHead>
                      <TableHead>Dari</TableHead>
                      <TableHead>Ke</TableHead>
                      <TableHead>Qty</TableHead>
                      <TableHead>Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {movements.map((movement) => (
                      <TableRow key={movement.id}>
                        <TableCell className="whitespace-nowrap text-sm">{formatWita(movement.posted_at)}</TableCell>
                        <TableCell>
                          <button
                            type="button"
                            className="cursor-pointer text-left font-medium hover:underline"
                            onClick={() => void openTransaction(movement.transaction_id)}
                          >
                            {movement.transaction_code}
                          </button>
                          <p className="text-xs text-muted-foreground">{txTypeLabels[movement.type as InventoryTransactionType] || movement.type}</p>
                        </TableCell>
                        <TableCell>
                          <p className="font-medium">{movement.sparepart_code}</p>
                          <p className="text-xs text-muted-foreground">{movement.sparepart_name}</p>
                        </TableCell>
                        <TableCell className="text-sm">{movement.from_location || "-"}</TableCell>
                        <TableCell className="text-sm">{movement.to_location || "-"}</TableCell>
                        <TableCell className="whitespace-nowrap">{movement.qty} {movement.uom}</TableCell>
                        <TableCell><TransactionStatusBadge status={movement.status} /></TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            <TablePagination
              page={movementPage}
              pageSize={movementPageSize}
              total={movementTotal}
              totalPages={movementPages}
              loading={movementLoading}
              itemLabel="pergerakan"
              onPageChange={setMovementPage}
              onPageSizeChange={(size) => { setMovementPageSize(size); setMovementPage(1); }}
            />
          </CardContent>
        </Card>
      ) : null}

      {tab === "opname" ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ClipboardList className="size-4" aria-hidden="true" /> Sesi stock opname
            </CardTitle>
            <CardDescription>
              Snapshot saldo per gudang, hitung fisik, lalu posting selisih sebagai adjustment.
            </CardDescription>
            <div className="grid gap-3 md:grid-cols-3">
              <Field>
                <FieldLabel htmlFor="opname-status">Status</FieldLabel>
                <NativeSelect
                  id="opname-status"
                  value={opnameStatus}
                  onChange={(event) => { setOpnameStatus(event.target.value); setOpnamePage(1); }}
                >
                  <NativeSelectOption value="">Semua status</NativeSelectOption>
                  {Object.entries(opnameStatusLabels).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
              <Field>
                <FieldLabel htmlFor="opname-warehouse">Gudang</FieldLabel>
                <NativeSelect
                  id="opname-warehouse"
                  value={opnameWarehouse}
                  onChange={(event) => { setOpnameWarehouse(event.target.value); setOpnamePage(1); }}
                >
                  <NativeSelectOption value="">Semua gudang</NativeSelectOption>
                  {warehouses.map((warehouse) => (
                    <NativeSelectOption key={warehouse.id} value={warehouse.id}>
                      {warehouse.code} - {warehouse.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </Field>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {opnameLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : opnameSessions.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon"><ClipboardList aria-hidden="true" /></EmptyMedia>
                  <EmptyTitle>Belum ada sesi opname</EmptyTitle>
                  <EmptyDescription>Buat sesi untuk memulai penghitungan fisik.</EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Kode</TableHead>
                      <TableHead>Gudang</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Progress</TableHead>
                      <TableHead>Dibuat</TableHead>
                      <TableHead>Posted</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {opnameSessions.map((session) => (
                      <TableRow key={session.id}>
                        <TableCell>
                          <button
                            type="button"
                            className="cursor-pointer text-left font-medium hover:underline"
                            onClick={() => void openOpname(session.id)}
                          >
                            {session.code}
                          </button>
                        </TableCell>
                        <TableCell>
                          <p>{session.warehouse_code}</p>
                          <p className="text-xs text-muted-foreground">{session.warehouse_name}</p>
                        </TableCell>
                        <TableCell><OpnameStatusBadge status={session.status} /></TableCell>
                        <TableCell className="whitespace-nowrap text-sm">
                          {session.counted_count} / {session.item_count} item
                        </TableCell>
                        <TableCell className="whitespace-nowrap text-sm">{formatWita(session.created_at)}</TableCell>
                        <TableCell className="whitespace-nowrap text-sm">
                          {session.posted_at ? formatWita(session.posted_at) : "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            <TablePagination
              page={opnamePage}
              pageSize={opnamePageSize}
              total={opnameTotal}
              totalPages={opnamePages}
              loading={opnameLoading}
              itemLabel="sesi"
              onPageChange={setOpnamePage}
              onPageSizeChange={(size) => { setOpnamePageSize(size); setOpnamePage(1); }}
            />
          </CardContent>
        </Card>
      ) : null}

      {tab === "master" ? (
        <div className="flex flex-col gap-4">
          {masterError ? (
            <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
              <CircleAlert aria-hidden="true" />
              <AlertTitle>Gagal</AlertTitle>
              <AlertDescription>{masterError}</AlertDescription>
            </Alert>
          ) : null}

          <div className="grid gap-4 xl:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Warehouse className="size-4" aria-hidden="true" /> Gudang
                </CardTitle>
                <CardDescription>Master gudang. Nonaktifkan alih-alih menghapus data.</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                {canMaster ? (
                  <form className="grid gap-3 rounded-lg border border-border p-3 md:grid-cols-3" onSubmit={submitWarehouse}>
                    <Field>
                      <FieldLabel htmlFor="warehouse-code">Kode</FieldLabel>
                      <Input id="warehouse-code" value={warehouseForm.code} maxLength={30} onChange={(event) => setWarehouseForm((current) => ({ ...current, code: event.target.value }))} required />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="warehouse-name">Nama</FieldLabel>
                      <Input id="warehouse-name" value={warehouseForm.name} maxLength={150} onChange={(event) => setWarehouseForm((current) => ({ ...current, name: event.target.value }))} required />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="warehouse-site">Site</FieldLabel>
                      <Input id="warehouse-site" value={warehouseForm.site_location} maxLength={150} onChange={(event) => setWarehouseForm((current) => ({ ...current, site_location: event.target.value }))} />
                    </Field>
                    <div className="md:col-span-3">
                      <Button type="submit" size="sm" disabled={masterBusy}>
                        {masterBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                        Tambah gudang
                      </Button>
                    </div>
                  </form>
                ) : null}
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Kode</TableHead>
                        <TableHead>Nama</TableHead>
                        <TableHead>Site</TableHead>
                        <TableHead>Jumlah lokasi</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Aksi</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {warehouses.map((warehouse) => (
                        <TableRow key={warehouse.id}>
                          <TableCell className="font-medium">{warehouse.code}</TableCell>
                          <TableCell>{warehouse.name}</TableCell>
                          <TableCell>{warehouse.site_location || "-"}</TableCell>
                          <TableCell>{warehouse.location_count}</TableCell>
                          <TableCell><ActiveBadge active={warehouse.is_active} /></TableCell>
                          <TableCell>
                            {canMaster ? (
                              <Button type="button" size="xs" variant="outline" onClick={() => void toggleWarehouse(warehouse)}>
                                {warehouse.is_active ? "Nonaktifkan" : "Aktifkan"}
                              </Button>
                            ) : null}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Boxes className="size-4" aria-hidden="true" /> Lokasi gudang
                </CardTitle>
                <CardDescription>Rack, bin, yard, atau staging di dalam gudang.</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                {canMaster ? (
                  <form className="grid gap-3 rounded-lg border border-border p-3 md:grid-cols-2" onSubmit={submitLocation}>
                    <Field>
                      <FieldLabel htmlFor="location-warehouse">Gudang</FieldLabel>
                      <NativeSelect id="location-warehouse" value={locationForm.warehouse_id} onChange={(event) => setLocationForm((current) => ({ ...current, warehouse_id: event.target.value }))} required>
                        <NativeSelectOption value="">Pilih gudang</NativeSelectOption>
                        {activeWarehouses.map((warehouse) => (
                          <NativeSelectOption key={warehouse.id} value={warehouse.id}>{warehouse.code} - {warehouse.name}</NativeSelectOption>
                        ))}
                      </NativeSelect>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="location-type">Tipe</FieldLabel>
                      <NativeSelect id="location-type" value={locationForm.location_type} onChange={(event) => setLocationForm((current) => ({ ...current, location_type: event.target.value }))}>
                        {Object.entries(locationTypeLabels).map(([value, label]) => (
                          <NativeSelectOption key={value} value={value}>{label}</NativeSelectOption>
                        ))}
                      </NativeSelect>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="location-code">Kode</FieldLabel>
                      <Input id="location-code" value={locationForm.code} maxLength={30} onChange={(event) => setLocationForm((current) => ({ ...current, code: event.target.value }))} required />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor="location-name">Nama</FieldLabel>
                      <Input id="location-name" value={locationForm.name} maxLength={150} onChange={(event) => setLocationForm((current) => ({ ...current, name: event.target.value }))} required />
                    </Field>
                    <div className="md:col-span-2">
                      <Button type="submit" size="sm" disabled={masterBusy}>
                        {masterBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <Plus aria-hidden="true" />}
                        Tambah lokasi
                      </Button>
                    </div>
                  </form>
                ) : null}
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Lokasi</TableHead>
                        <TableHead>Gudang</TableHead>
                        <TableHead>Nama</TableHead>
                        <TableHead>Tipe</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Aksi</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {locations.map((location) => (
                        <TableRow key={location.id}>
                          <TableCell className="font-medium">{location.code}</TableCell>
                          <TableCell>{location.warehouse_code}</TableCell>
                          <TableCell>{location.name}</TableCell>
                          <TableCell>{locationTypeLabels[location.location_type] || location.location_type}</TableCell>
                          <TableCell><ActiveBadge active={location.is_active} /></TableCell>
                          <TableCell>
                            {canMaster ? (
                              <Button type="button" size="xs" variant="outline" onClick={() => void toggleLocation(location)}>
                                {location.is_active ? "Nonaktifkan" : "Aktifkan"}
                              </Button>
                            ) : null}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="size-4" aria-hidden="true" /> Minimum stock policy
              </CardTitle>
              <CardDescription>
                Policy per sparepart dan lokasi. Status low/critical dihitung dari available qty.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              {canMaster ? (
                <form className="grid gap-3 rounded-lg border border-border p-3 md:grid-cols-3 xl:grid-cols-6" onSubmit={submitPolicy}>
                  <Field className="xl:col-span-2">
                    <FieldLabel htmlFor="policy-sparepart">Sparepart</FieldLabel>
                    <NativeSelect
                      id="policy-sparepart"
                      value={policyForm.sparepart_id}
                      disabled={policyEditingID !== null}
                      onChange={(event) => setPolicyForm((current) => ({ ...current, sparepart_id: event.target.value }))}
                      required
                    >
                      <NativeSelectOption value="">Pilih sparepart</NativeSelectOption>
                      {activeSpareparts.map((sparepart) => (
                        <NativeSelectOption key={sparepart.id} value={sparepart.id}>{sparepartLabel(sparepart)}</NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </Field>
                  <Field className="xl:col-span-2">
                    <FieldLabel htmlFor="policy-location">Lokasi</FieldLabel>
                    <NativeSelect
                      id="policy-location"
                      value={policyForm.warehouse_location_id}
                      disabled={policyEditingID !== null}
                      onChange={(event) => setPolicyForm((current) => ({ ...current, warehouse_location_id: event.target.value }))}
                      required
                    >
                      <NativeSelectOption value="">Pilih lokasi</NativeSelectOption>
                      {activeLocations.map((location) => (
                        <NativeSelectOption key={location.id} value={location.id}>{locationLabel(location)}</NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="policy-min">Min qty</FieldLabel>
                    <Input id="policy-min" type="number" min={0} step="0.001" value={policyForm.min_qty} onChange={(event) => setPolicyForm((current) => ({ ...current, min_qty: event.target.value }))} required />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="policy-critical">Critical qty</FieldLabel>
                    <Input id="policy-critical" type="number" min={0} step="0.001" value={policyForm.critical_qty} onChange={(event) => setPolicyForm((current) => ({ ...current, critical_qty: event.target.value }))} required />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="policy-reorder">Reorder qty</FieldLabel>
                    <Input id="policy-reorder" type="number" min={0} step="0.001" value={policyForm.reorder_qty} onChange={(event) => setPolicyForm((current) => ({ ...current, reorder_qty: event.target.value }))} />
                  </Field>
                  <label className="flex items-center gap-2 text-sm xl:col-span-2">
                    <input type="checkbox" checked={policyForm.is_active} onChange={(event) => setPolicyForm((current) => ({ ...current, is_active: event.target.checked }))} className="size-4 rounded border-border" />
                    Policy aktif
                  </label>
                  <div className="flex items-end gap-2 xl:col-span-2">
                    <Button type="submit" size="sm" disabled={masterBusy}>
                      {masterBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                      {policyEditingID ? "Simpan perubahan" : "Simpan policy"}
                    </Button>
                    {policyEditingID ? (
                      <Button type="button" size="sm" variant="outline" onClick={resetPolicyForm}>Batal</Button>
                    ) : null}
                  </div>
                </form>
              ) : null}
              {policyLoading ? (
                <div className="flex flex-col gap-2">
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-10 w-full" />
                </div>
              ) : policies.length === 0 ? (
                <p className="text-sm text-muted-foreground">Belum ada stock policy.</p>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Sparepart</TableHead>
                        <TableHead>Lokasi</TableHead>
                        <TableHead>Min</TableHead>
                        <TableHead>Critical</TableHead>
                        <TableHead>Reorder</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Aksi</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {policies.map((policy) => (
                        <TableRow key={policy.id}>
                          <TableCell>
                            <p className="font-medium">{policy.sparepart_code}</p>
                            <p className="text-xs text-muted-foreground">{policy.sparepart_name}</p>
                          </TableCell>
                          <TableCell>
                            <p>{policy.warehouse_code} / {policy.location_code}</p>
                            <p className="text-xs text-muted-foreground">{policy.location_name}</p>
                          </TableCell>
                          <TableCell>{policy.min_qty}</TableCell>
                          <TableCell>{policy.critical_qty}</TableCell>
                          <TableCell>{policy.reorder_qty ?? "-"}</TableCell>
                          <TableCell><ActiveBadge active={policy.is_active} /></TableCell>
                          <TableCell>
                            {canMaster ? (
                              <Button type="button" size="xs" variant="outline" onClick={() => editPolicy(policy)}>
                                Edit
                              </Button>
                            ) : null}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
              <TablePagination
                page={policyPage}
                pageSize={policyPageSize}
                total={policyTotal}
                totalPages={policyPages}
                loading={policyLoading}
                itemLabel="policy"
                onPageChange={setPolicyPage}
                onPageSizeChange={(size) => { setPolicyPageSize(size); setPolicyPage(1); }}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Scale className="size-4" aria-hidden="true" /> Rekonsiliasi ledger
              </CardTitle>
              <CardDescription>Bandingkan saldo materialized dengan jumlah ledger posted.</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <div className="flex items-center gap-2">
                <Button type="button" variant="outline" onClick={() => void checkReconciliation()} disabled={reconciliationLoading}>
                  {reconciliationLoading ? <Loader2 className="animate-spin" aria-hidden="true" /> : <RefreshCw aria-hidden="true" />}
                  Jalankan rekonsiliasi
                </Button>
                {reconciliation ? (
                  <Badge variant={reconciliation.mismatch === 0 ? "secondary" : "destructive"}>
                    {reconciliation.mismatch === 0 ? "Seimbang" : `${reconciliation.mismatch} selisih`}
                  </Badge>
                ) : null}
              </div>
              {reconciliation && reconciliation.rows.length > 0 ? (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Sparepart</TableHead>
                        <TableHead>Lokasi</TableHead>
                        <TableHead>Saldo</TableHead>
                        <TableHead>Ledger</TableHead>
                        <TableHead>Selisih</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {reconciliation.rows.map((row) => (
                        <TableRow key={`${row.sparepart_id}-${row.warehouse_location_id}`}>
                          <TableCell>
                            <p className="font-medium">{row.sparepart_code}</p>
                            <p className="text-xs text-muted-foreground">{row.sparepart_name}</p>
                          </TableCell>
                          <TableCell>{row.warehouse_code} / {row.location_code}</TableCell>
                          <TableCell>{row.balance_on_hand}</TableCell>
                          <TableCell>{row.ledger_on_hand}</TableCell>
                          <TableCell className={cn(row.difference !== 0 && "font-medium text-destructive")}>{row.difference}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : reconciliation ? (
                <p className="text-sm text-muted-foreground">Tidak ada selisih. Diperiksa {formatWita(reconciliation.checked_at)}.</p>
              ) : null}
            </CardContent>
          </Card>
        </div>
      ) : null}

      <Sheet open={txFormOpen} onOpenChange={setTxFormOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-4xl">
          <SheetHeader>
            <SheetTitle>Catat {txTypeLabels[txFormType]}</SheetTitle>
            <SheetDescription>
              Transaksi langsung posted. Gunakan kunci idempotensi yang sama bila retry setelah gangguan jaringan.
            </SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitTx}>
            <div className="grid gap-3 sm:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="tx-reference">Nomor referensi</FieldLabel>
                <Input
                  id="tx-reference"
                  value={txForm.reference_no}
                  maxLength={100}
                  onChange={(event) => setTxForm((current) => ({ ...current, reference_no: event.target.value }))}
                  placeholder="DO, BA, atau referensi manual"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="tx-idempotency">Kunci idempotensi</FieldLabel>
                <Input id="tx-idempotency" value={txForm.idempotency_key} readOnly className="font-mono text-xs" />
              </Field>
            </div>
            <Field>
              <FieldLabel htmlFor="tx-notes">
                Catatan {txFormType === "issue" || txFormType === "adjustment" ? "(wajib)" : ""}
              </FieldLabel>
              <Textarea
                id="tx-notes"
                value={txForm.notes}
                maxLength={1000}
                onChange={(event) => setTxForm((current) => ({ ...current, notes: event.target.value }))}
                rows={3}
                required={txFormType === "issue" || txFormType === "adjustment"}
                placeholder={txFormType === "adjustment" ? "Alasan adjustment..." : "Catatan transaksi..."}
              />
            </Field>

            <div className="flex flex-col gap-3">
              {txForm.items.map((item, index) => (
                <div key={item.key} className="flex flex-col gap-3 rounded-lg border border-border p-3">
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm font-medium">Item {index + 1}</p>
                    <Button
                      type="button"
                      size="xs"
                      variant="outline"
                      disabled={txForm.items.length === 1}
                      onClick={() => removeTxItem(item.key)}
                    >
                      <Trash2 aria-hidden="true" /> Hapus
                    </Button>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    <Field className="md:col-span-2">
                      <FieldLabel htmlFor={`tx-item-sparepart-${item.key}`}>Sparepart</FieldLabel>
                      <NativeSelect
                        id={`tx-item-sparepart-${item.key}`}
                        value={item.sparepart_id}
                        onChange={(event) => updateTxItem(item.key, { sparepart_id: event.target.value })}
                        required
                      >
                        <NativeSelectOption value="">Pilih sparepart</NativeSelectOption>
                        {activeSpareparts.map((sparepart) => (
                          <NativeSelectOption key={sparepart.id} value={sparepart.id}>
                            {sparepartLabel(sparepart)} ({sparepart.uom})
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                    </Field>

                    {txFormType === "transfer" ? (
                      <>
                        <Field>
                          <FieldLabel htmlFor={`tx-item-from-${item.key}`}>Lokasi asal</FieldLabel>
                          <NativeSelect
                            id={`tx-item-from-${item.key}`}
                            value={item.from_location_id}
                            onChange={(event) => updateTxItem(item.key, { from_location_id: event.target.value })}
                            required
                          >
                            <NativeSelectOption value="">Pilih lokasi asal</NativeSelectOption>
                            {activeLocations.map((location) => (
                              <NativeSelectOption key={location.id} value={location.id}>{locationLabel(location)}</NativeSelectOption>
                            ))}
                          </NativeSelect>
                        </Field>
                        <Field>
                          <FieldLabel htmlFor={`tx-item-to-${item.key}`}>Lokasi tujuan</FieldLabel>
                          <NativeSelect
                            id={`tx-item-to-${item.key}`}
                            value={item.to_location_id}
                            onChange={(event) => updateTxItem(item.key, { to_location_id: event.target.value })}
                            required
                          >
                            <NativeSelectOption value="">Pilih lokasi tujuan</NativeSelectOption>
                            {activeLocations.map((location) => (
                              <NativeSelectOption key={location.id} value={location.id}>{locationLabel(location)}</NativeSelectOption>
                            ))}
                          </NativeSelect>
                        </Field>
                      </>
                    ) : (
                      <>
                        {txFormType === "adjustment" ? (
                          <Field>
                            <FieldLabel htmlFor={`tx-item-direction-${item.key}`}>Arah</FieldLabel>
                            <NativeSelect
                              id={`tx-item-direction-${item.key}`}
                              value={item.direction}
                              onChange={(event) => updateTxItem(item.key, { direction: event.target.value as "increase" | "decrease" })}
                            >
                              <NativeSelectOption value="increase">Tambah stok</NativeSelectOption>
                              <NativeSelectOption value="decrease">Kurangi stok</NativeSelectOption>
                            </NativeSelect>
                          </Field>
                        ) : null}
                        <Field>
                          <FieldLabel htmlFor={`tx-item-location-${item.key}`}>
                            {txFormType === "issue" ? "Lokasi asal" : "Lokasi tujuan"}
                          </FieldLabel>
                          <NativeSelect
                            id={`tx-item-location-${item.key}`}
                            value={item.location_id}
                            onChange={(event) => updateTxItem(item.key, { location_id: event.target.value })}
                            required
                          >
                            <NativeSelectOption value="">Pilih lokasi</NativeSelectOption>
                            {activeLocations.map((location) => (
                              <NativeSelectOption key={location.id} value={location.id}>{locationLabel(location)}</NativeSelectOption>
                            ))}
                          </NativeSelect>
                        </Field>
                      </>
                    )}

                    <Field>
                      <FieldLabel htmlFor={`tx-item-qty-${item.key}`}>Qty</FieldLabel>
                      <Input
                        id={`tx-item-qty-${item.key}`}
                        type="number"
                        min={0.001}
                        step="0.001"
                        value={item.qty}
                        onChange={(event) => updateTxItem(item.key, { qty: event.target.value })}
                        required
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor={`tx-item-notes-${item.key}`}>Catatan item</FieldLabel>
                      <Input
                        id={`tx-item-notes-${item.key}`}
                        value={item.notes}
                        maxLength={500}
                        onChange={(event) => updateTxItem(item.key, { notes: event.target.value })}
                      />
                    </Field>
                  </div>
                </div>
              ))}
            </div>

            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => setTxForm((current) => ({ ...current, items: [...current.items, emptyTxItem()] }))}
            >
              <Plus aria-hidden="true" /> Tambah item
            </Button>

            {txFormError ? <FieldError>{txFormError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" disabled={txFormBusy}>
                {txFormBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                Posting {txTypeLabels[txFormType]}
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <Sheet open={txDetailOpen} onOpenChange={setTxDetailOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-3xl">
          <SheetHeader>
            <SheetTitle>{txDetail ? txDetail.code : "Detail transaksi"}</SheetTitle>
            <SheetDescription>
              {txDetail ? `${txTypeLabels[txDetail.type as InventoryTransactionType] || txDetail.type} · ${formatWita(txDetail.posted_at)}` : "Memuat detail..."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex flex-col gap-4 px-4 pb-4">
            {txDetailError ? <FieldError>{txDetailError}</FieldError> : null}
            {txDetailLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : txDetail ? (
              <>
                <div className="grid gap-3 rounded-lg border border-border p-3 text-sm sm:grid-cols-2">
                  <p><span className="text-muted-foreground">Status:</span> <TransactionStatusBadge status={txDetail.status} /></p>
                  <p><span className="text-muted-foreground">Posted by:</span> {txDetail.posted_by_name || "-"}</p>
                  <p><span className="text-muted-foreground">Referensi:</span> {txDetail.reference_no || "-"}</p>
                  <p><span className="text-muted-foreground">Tipe referensi:</span> {txDetail.reference_type || "-"}</p>
                  <p className="sm:col-span-2"><span className="text-muted-foreground">Catatan:</span> {txDetail.notes || "-"}</p>
                  <p className="sm:col-span-2 break-all"><span className="text-muted-foreground">Idempotency:</span> <span className="font-mono text-xs">{txDetail.idempotency_key || "-"}</span></p>
                </div>

                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Sparepart</TableHead>
                        <TableHead>Dari</TableHead>
                        <TableHead>Ke</TableHead>
                        <TableHead>Qty</TableHead>
                        <TableHead>Catatan</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {txDetail.items.map((item) => (
                        <TableRow key={item.id}>
                          <TableCell>
                            <p className="font-medium">{item.sparepart_code}</p>
                            <p className="text-xs text-muted-foreground">{item.sparepart_name}</p>
                          </TableCell>
                          <TableCell className="text-sm">{item.from_location || "-"}</TableCell>
                          <TableCell className="text-sm">{item.to_location || "-"}</TableCell>
                          <TableCell className="whitespace-nowrap">{item.qty} {item.uom}</TableCell>
                          <TableCell className="max-w-56 text-sm text-muted-foreground">{item.notes || "-"}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {canApprove && txDetail.status === "posted" && !txDetail.reversal_of_id ? (
                  <form className="flex flex-col gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3" onSubmit={submitReverse}>
                    <p className="flex items-center gap-2 text-sm font-medium">
                      <RotateCcw className="size-4" aria-hidden="true" /> Reversal transaksi
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Reversal membuat transaksi cermin. Transaksi asli tetap tersimpan dan tidak dapat diedit.
                    </p>
                    <Field>
                      <FieldLabel htmlFor="reverse-reason">Alasan reversal</FieldLabel>
                      <Textarea
                        id="reverse-reason"
                        value={reverseReason}
                        maxLength={1000}
                        onChange={(event) => setReverseReason(event.target.value)}
                        rows={3}
                        required
                      />
                    </Field>
                    <Button type="submit" variant="destructive" size="sm" disabled={reverseBusy}>
                      {reverseBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <RotateCcw aria-hidden="true" />}
                      Posting reversal
                    </Button>
                  </form>
                ) : txDetail.status === "reversed" ? (
                  <Alert>
                    <RotateCcw aria-hidden="true" />
                    <AlertTitle>Transaksi sudah direversal</AlertTitle>
                    <AlertDescription>Gunakan transaksi baru untuk koreksi berikutnya.</AlertDescription>
                  </Alert>
                ) : null}
              </>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

      <Sheet open={opnameCreateOpen} onOpenChange={setOpnameCreateOpen}>
        <SheetContent className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>Buat sesi stock opname</SheetTitle>
            <SheetDescription>
              Snapshot saldo seluruh lokasi aktif pada gudang terpilih akan dibuat.
            </SheetDescription>
          </SheetHeader>
          <form className="flex flex-col gap-4 px-4 pb-4" onSubmit={submitOpnameCreate}>
            <Field>
              <FieldLabel htmlFor="opname-create-warehouse">Gudang</FieldLabel>
              <NativeSelect
                id="opname-create-warehouse"
                value={opnameCreateWarehouse}
                onChange={(event) => setOpnameCreateWarehouse(event.target.value)}
                required
              >
                <NativeSelectOption value="">Pilih gudang</NativeSelectOption>
                {activeWarehouses.map((warehouse) => (
                  <NativeSelectOption key={warehouse.id} value={warehouse.id}>
                    {warehouse.code} - {warehouse.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field>
              <FieldLabel htmlFor="opname-create-notes">Catatan</FieldLabel>
              <Textarea
                id="opname-create-notes"
                value={opnameCreateNotes}
                maxLength={1000}
                onChange={(event) => setOpnameCreateNotes(event.target.value)}
                rows={3}
              />
            </Field>
            {opnameCreateError ? <FieldError>{opnameCreateError}</FieldError> : null}
            <SheetFooter className="px-0">
              <Button type="submit" disabled={opnameCreateBusy}>
                {opnameCreateBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <ClipboardCheck aria-hidden="true" />}
                Buat dan mulai hitung
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>

      <Sheet open={opnameDetailOpen} onOpenChange={setOpnameDetailOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-5xl">
          <SheetHeader>
            <SheetTitle>{opnameDetail ? opnameDetail.code : "Detail stock opname"}</SheetTitle>
            <SheetDescription>
              {opnameDetail ? `${opnameDetail.warehouse_code} - ${opnameDetail.warehouse_name}` : "Memuat detail..."}
            </SheetDescription>
          </SheetHeader>
          <div className="flex flex-col gap-4 px-4 pb-4">
            {opnameDetailError ? <FieldError>{opnameDetailError}</FieldError> : null}
            {opnameDetailLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : opnameDetail ? (
              <>
                <div className="flex flex-wrap items-center gap-3 rounded-lg border border-border p-3 text-sm">
                  <OpnameStatusBadge status={opnameDetail.status} />
                  <span>{opnameDetail.counted_count} / {opnameDetail.item_count} item dihitung</span>
                  <span className="text-muted-foreground">Dibuat {formatWita(opnameDetail.created_at)}</span>
                  {opnameDetail.posted_at ? <span className="text-muted-foreground">Posted {formatWita(opnameDetail.posted_at)}</span> : null}
                  {opnameDetail.notes ? <span className="text-muted-foreground">{opnameDetail.notes}</span> : null}
                </div>

                {opnameDetail.cancel_reason ? (
                  <Alert>
                    <CircleAlert aria-hidden="true" />
                    <AlertTitle>Sesi dibatalkan</AlertTitle>
                    <AlertDescription>{opnameDetail.cancel_reason}</AlertDescription>
                  </Alert>
                ) : null}

                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Sparepart</TableHead>
                        <TableHead>Lokasi</TableHead>
                        <TableHead>System qty</TableHead>
                        <TableHead>Hitung fisik</TableHead>
                        <TableHead>Selisih</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {opnameDetail.items.map((item) => {
                        const raw = opnameCounts[item.id] ?? "";
                        const parsed = raw.trim() === "" ? null : Number(raw);
                        const difference = parsed !== null && Number.isFinite(parsed)
                          ? parsed - item.system_qty
                          : item.difference ?? null;
                        const editable = canCount && (opnameDetail.status === "counting" || opnameDetail.status === "review");
                        return (
                          <TableRow key={item.id}>
                            <TableCell>
                              <p className="font-medium">{item.sparepart_code}</p>
                              <p className="text-xs text-muted-foreground">{item.sparepart_name}</p>
                            </TableCell>
                            <TableCell>
                              <p>{item.location_code}</p>
                              <p className="text-xs text-muted-foreground">{item.location_name}</p>
                            </TableCell>
                            <TableCell className="whitespace-nowrap">{item.system_qty} {item.uom}</TableCell>
                            <TableCell className="min-w-32">
                              {editable ? (
                                <Input
                                  type="number"
                                  min={0}
                                  step="0.001"
                                  value={raw}
                                  aria-label={`Qty hitung ${item.sparepart_code} di ${item.location_code}`}
                                  onChange={(event) => setOpnameCounts((current) => ({ ...current, [item.id]: event.target.value }))}
                                />
                              ) : item.counted_qty === undefined ? "-" : `${item.counted_qty} ${item.uom}`}
                            </TableCell>
                            <TableCell className={cn("whitespace-nowrap", difference !== null && difference !== 0 && "font-medium text-destructive")}>
                              {difference === null ? "-" : difference}
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  {canCount && (opnameDetail.status === "counting" || opnameDetail.status === "review") ? (
                    <Button type="button" size="sm" onClick={() => void saveCounts()} disabled={opnameActionBusy}>
                      {opnameActionBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
                      Simpan hasil hitung
                    </Button>
                  ) : null}
                  {canApprove && (opnameDetail.status === "counting" || opnameDetail.status === "review") ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => void postOpname()}
                      disabled={opnameActionBusy || opnameDetail.counted_count !== opnameDetail.item_count || opnameDetail.item_count === 0 || opnameHasUnsavedCounts}
                    >
                      {opnameActionBusy ? <Loader2 className="animate-spin" aria-hidden="true" /> : <ShieldCheck aria-hidden="true" />}
                      Posting opname
                    </Button>
                  ) : null}
                  {opnameHasUnsavedCounts ? (
                    <span className="text-xs text-muted-foreground">Simpan hasil hitung sebelum posting.</span>
                  ) : null}
                </div>

                {canApprove && (opnameDetail.status === "draft" || opnameDetail.status === "counting" || opnameDetail.status === "review") ? (
                  <div className="flex flex-col gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3">
                    <p className="text-sm font-medium">Batalkan sesi</p>
                    <Input
                      value={opnameCancelReason}
                      maxLength={1000}
                      onChange={(event) => setOpnameCancelReason(event.target.value)}
                      placeholder="Alasan pembatalan..."
                      aria-label="Alasan pembatalan opname"
                    />
                    <Button type="button" size="sm" variant="destructive" onClick={() => void cancelOpname()} disabled={opnameActionBusy}>
                      Batalkan sesi
                    </Button>
                  </div>
                ) : null}
              </>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function StockStatusBadge({ status }: { status: string }) {
  if (status === "critical") {
    return <Badge variant="destructive">{stockStatusLabels[status]}</Badge>;
  }
  if (status === "low") {
    return (
      <Badge variant="secondary" className="bg-amber-500/10 text-amber-700 dark:text-amber-300">
        {stockStatusLabels[status]}
      </Badge>
    );
  }
  if (status === "ok") {
    return (
      <Badge variant="secondary" className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300">
        {stockStatusLabels[status]}
      </Badge>
    );
  }
  return <Badge variant="outline">{stockStatusLabels[status] || status}</Badge>;
}

function TransactionStatusBadge({ status }: { status: string }) {
  if (status === "reversed") {
    return <Badge variant="outline" className="text-muted-foreground">Reversed</Badge>;
  }
  return (
    <Badge variant="secondary" className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300">
      Posted
    </Badge>
  );
}

function OpnameStatusBadge({ status }: { status: string }) {
  if (status === "posted") {
    return (
      <Badge variant="secondary" className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-300">
        Posted
      </Badge>
    );
  }
  if (status === "cancelled") {
    return <Badge variant="destructive">Dibatalkan</Badge>;
  }
  if (status === "review") {
    return (
      <Badge variant="secondary" className="bg-sky-500/10 text-sky-700 dark:text-sky-300">
        Review
      </Badge>
    );
  }
  if (status === "counting") {
    return (
      <Badge variant="secondary" className="bg-amber-500/10 text-amber-700 dark:text-amber-300">
        Counting
      </Badge>
    );
  }
  return <Badge variant="outline">{opnameStatusLabels[status] || status}</Badge>;
}
