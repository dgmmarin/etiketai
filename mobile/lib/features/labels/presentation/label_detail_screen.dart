import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../core/theme/app_theme.dart';
import '../data/labels_repository.dart';
import '../domain/label_detail.dart';
import '../domain/label_status.dart';
import 'label_status_helpers.dart';
import 'labels_provider.dart';

class LabelDetailScreen extends ConsumerStatefulWidget {
  const LabelDetailScreen({super.key, required this.labelId});

  final String labelId;

  @override
  ConsumerState<LabelDetailScreen> createState() => _LabelDetailScreenState();
}

class _LabelDetailScreenState extends ConsumerState<LabelDetailScreen> {
  LabelDetail? _detail;
  bool _isLoading = true;
  String? _error;

  // Editing
  final Map<String, TextEditingController> _controllers = {};
  bool _isDirty = false;
  bool _isSaving = false;
  bool _isConfirming = false;

  // Polling
  Timer? _pollTimer;

  @override
  void initState() {
    super.initState();
    _loadDetail();
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    for (final c in _controllers.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _loadDetail() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });
    try {
      final repo = ref.read(labelsRepositoryProvider);
      final detail = await repo.get(widget.labelId);
      _applyDetail(detail);
    } catch (e) {
      setState(() => _error = 'Nu s-au putut încărca detaliile etichetei.');
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  void _applyDetail(LabelDetail detail) {
    if (!mounted) return;
    setState(() {
      _detail = detail;
      for (final c in _controllers.values) {
        c.dispose();
      }
      _controllers.clear();
      for (final f in detail.fields) {
        _controllers[f.fieldKey] = TextEditingController(
          text: f.translatedText ?? '',
        );
      }
      _isDirty = false;
    });

    if (detail.status == LabelStatusValue.processing ||
        detail.status == LabelStatusValue.pending) {
      _startPolling();
    } else {
      _pollTimer?.cancel();
    }
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 3), (_) async {
      if (!mounted) return;
      try {
        final repo = ref.read(labelsRepositoryProvider);
        final status = await repo.getStatus(widget.labelId);
        if (status.status != LabelStatusValue.processing &&
            status.status != LabelStatusValue.pending) {
          _pollTimer?.cancel();
          _loadDetail();
        }
      } catch (_) {}
    });
  }

  Future<void> _save() async {
    if (!_isDirty) return;
    setState(() => _isSaving = true);
    try {
      final repo = ref.read(labelsRepositoryProvider);
      final updatedFields = {
        for (final entry in _controllers.entries) entry.key: entry.value.text,
      };
      final updated = await repo.updateFields(widget.labelId, updatedFields);
      _applyDetail(updated);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Modificările au fost salvate.')),
        );
      }
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Eroare la salvare.')),
        );
      }
    } finally {
      if (mounted) setState(() => _isSaving = false);
    }
  }

  Future<void> _confirm() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Confirmați eticheta?'),
        content: const Text(
          'După confirmare eticheta nu mai poate fi editată.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Anulare'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Confirmați'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    setState(() => _isConfirming = true);
    try {
      final repo = ref.read(labelsRepositoryProvider);
      final updated = await repo.confirm(widget.labelId);
      _applyDetail(updated);
      ref.invalidate(labelsProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
              content: Text('Eticheta a fost confirmată cu succes.')),
        );
      }
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Eroare la confirmare.')),
        );
      }
    } finally {
      if (mounted) setState(() => _isConfirming = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_isLoading) {
      return Scaffold(
        appBar: AppBar(title: const Text('Detalii etichetă')),
        body: const Center(child: CircularProgressIndicator()),
      );
    }

    if (_error != null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Detalii etichetă')),
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(_error!, style: const TextStyle(color: Colors.black54)),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: _loadDetail,
                child: const Text('Reîncercați'),
              ),
            ],
          ),
        ),
      );
    }

    final detail = _detail!;
    final isEditable = detail.status == LabelStatusValue.ready;
    final isProcessing = detail.status == LabelStatusValue.processing ||
        detail.status == LabelStatusValue.pending;

    return Scaffold(
      appBar: AppBar(
        title: Text(detail.productName ?? 'Etichetă'),
        actions: [
          if (isEditable && _isDirty)
            TextButton(
              onPressed: _isSaving ? null : _save,
              child: _isSaving
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white),
                    )
                  : const Text('Salvează',
                      style: TextStyle(color: Colors.white)),
            ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Image
            if (detail.imageUrl != null)
              ClipRRect(
                borderRadius: BorderRadius.circular(12),
                child: CachedNetworkImage(
                  imageUrl: detail.imageUrl!,
                  width: double.infinity,
                  height: 220,
                  fit: BoxFit.cover,
                  placeholder: (_, __) => Container(
                    height: 220,
                    color: Colors.grey[200],
                    child: const Center(child: CircularProgressIndicator()),
                  ),
                  errorWidget: (_, __, ___) => Container(
                    height: 220,
                    color: Colors.grey[200],
                    child: const Icon(Icons.broken_image_outlined,
                        size: 60, color: Colors.black26),
                  ),
                ),
              ),

            const SizedBox(height: 16),

            // Status row
            Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 12, vertical: 4),
                  decoration: BoxDecoration(
                    color: statusColor(detail.status),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: Text(
                    statusLabel(detail.status),
                    style: const TextStyle(fontWeight: FontWeight.w600),
                  ),
                ),
                if (isProcessing) ...[
                  const SizedBox(width: 12),
                  const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                  const SizedBox(width: 8),
                  const Text('Procesare AI...',
                      style: TextStyle(color: Colors.black54, fontSize: 13)),
                ],
              ],
            ),

            const SizedBox(height: 8),

            // Dates & metadata
            Text(
              'Creat: ${DateFormat('dd.MM.yyyy HH:mm', 'ro_RO').format(detail.createdAt.toLocal())}',
              style: const TextStyle(color: Colors.black45, fontSize: 12),
            ),
            if (detail.sourceLanguage != null)
              Text(
                'Limbă sursă: ${detail.sourceLanguage}',
                style: const TextStyle(color: Colors.black45, fontSize: 12),
              ),

            const SizedBox(height: 20),

            // Translated fields
            if (detail.fields.isNotEmpty) ...[
              Text(
                'Câmpuri traduse',
                style: Theme.of(context)
                    .textTheme
                    .titleMedium
                    ?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              ...detail.fields.map(
                (f) => _FieldCard(
                  field: f,
                  controller: _controllers[f.fieldKey]!,
                  editable: isEditable,
                  onChanged: () => setState(() => _isDirty = true),
                ),
              ),
            ],

            const SizedBox(height: 20),

            // Compliance
            if (detail.complianceCheck != null)
              _ComplianceCard(check: detail.complianceCheck!),

            const SizedBox(height: 24),

            // Action buttons
            if (isEditable) ...[
              if (_isDirty)
                ElevatedButton.icon(
                  onPressed: _isSaving ? null : _save,
                  icon: const Icon(Icons.save_outlined),
                  label: const Text('Salvați modificările'),
                ),
              const SizedBox(height: 12),
              OutlinedButton.icon(
                onPressed: _isConfirming ? null : _confirm,
                icon: _isConfirming
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.check_circle_outline),
                label: const Text('Confirmați eticheta'),
                style: OutlinedButton.styleFrom(
                  minimumSize: const Size(double.infinity, 48),
                  foregroundColor: AppColors.secondary,
                  side: const BorderSide(color: AppColors.secondary),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8),
                  ),
                ),
              ),
            ],

            if (detail.status == LabelStatusValue.confirmed &&
                detail.confirmedAt != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Row(
                  children: [
                    const Icon(Icons.check_circle,
                        color: AppColors.secondary, size: 18),
                    const SizedBox(width: 6),
                    Text(
                      'Confirmat la ${DateFormat('dd.MM.yyyy HH:mm', 'ro_RO').format(detail.confirmedAt!.toLocal())}',
                      style: const TextStyle(
                          color: AppColors.secondary, fontSize: 13),
                    ),
                  ],
                ),
              ),

            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Field card
// ---------------------------------------------------------------------------

class _FieldCard extends StatelessWidget {
  const _FieldCard({
    required this.field,
    required this.controller,
    required this.editable,
    required this.onChanged,
  });

  final LabelField field;
  final TextEditingController controller;
  final bool editable;
  final VoidCallback onChanged;

  String _friendlyKey(String key) {
    const map = <String, String>{
      'product_name': 'Denumire produs',
      'ingredients': 'Ingrediente',
      'net_weight': 'Greutate netă',
      'manufacturer': 'Producător',
      'country_of_origin': 'Țara de origine',
      'expiry_info': 'Valabilitate',
      'storage_conditions': 'Condiții de depozitare',
      'allergens': 'Alergeni',
      'nutritional_info': 'Informații nutriționale',
      'instructions': 'Instrucțiuni de utilizare',
    };
    return map[key] ?? key.replaceAll('_', ' ').toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              _friendlyKey(field.fieldKey),
              style: const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: Colors.black54,
                letterSpacing: 0.5,
              ),
            ),
            if (field.originalText != null) ...[
              const SizedBox(height: 4),
              Text(
                'Original: ${field.originalText}',
                style: const TextStyle(
                    fontSize: 12,
                    color: Colors.black38,
                    fontStyle: FontStyle.italic),
              ),
            ],
            const SizedBox(height: 6),
            editable
                ? TextFormField(
                    controller: controller,
                    maxLines: null,
                    onChanged: (_) => onChanged(),
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      isDense: true,
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    ),
                    style: const TextStyle(fontSize: 14),
                  )
                : Text(
                    controller.text.isEmpty ? '—' : controller.text,
                    style: const TextStyle(fontSize: 14),
                  ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Compliance card
// ---------------------------------------------------------------------------

class _ComplianceCard extends StatelessWidget {
  const _ComplianceCard({required this.check});

  final ComplianceCheck check;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: check.isCompliant
          ? AppColors.statusReady
          : AppColors.statusFailed.withValues(alpha: 0.5),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  check.isCompliant
                      ? Icons.verified_outlined
                      : Icons.warning_amber_outlined,
                  color: check.isCompliant
                      ? AppColors.secondary
                      : AppColors.warning,
                ),
                const SizedBox(width: 8),
                Text(
                  check.isCompliant
                      ? 'Conformitate OK'
                      : 'Probleme de conformitate',
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
              ],
            ),
            if (check.issues.isNotEmpty) ...[
              const SizedBox(height: 10),
              const Text('Erori:',
                  style: TextStyle(
                      fontWeight: FontWeight.w600, color: AppColors.error)),
              ...check.issues.map(
                (i) => Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Icon(Icons.cancel_outlined,
                          size: 14, color: AppColors.error),
                      const SizedBox(width: 6),
                      Expanded(
                          child: Text(i, style: const TextStyle(fontSize: 13))),
                    ],
                  ),
                ),
              ),
            ],
            if (check.warnings.isNotEmpty) ...[
              const SizedBox(height: 10),
              const Text('Avertismente:',
                  style: TextStyle(
                      fontWeight: FontWeight.w600, color: AppColors.warning)),
              ...check.warnings.map(
                (w) => Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Icon(Icons.warning_outlined,
                          size: 14, color: AppColors.warning),
                      const SizedBox(width: 6),
                      Expanded(
                          child: Text(w,
                              style: const TextStyle(fontSize: 13))),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
