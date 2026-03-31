import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/theme/app_theme.dart';
import '../domain/label.dart';
import '../domain/label_status.dart';
import 'label_status_helpers.dart';
import 'labels_provider.dart';

class LabelsScreen extends ConsumerStatefulWidget {
  const LabelsScreen({super.key});

  @override
  ConsumerState<LabelsScreen> createState() => _LabelsScreenState();
}

class _LabelsScreenState extends ConsumerState<LabelsScreen> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(labelsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Etichete'),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_outline),
            onPressed: () => context.go('/workspace'),
          ),
        ],
      ),
      body: Column(
        children: [
          // Search bar
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: TextField(
              controller: _searchCtrl,
              decoration: InputDecoration(
                hintText: 'Căutați etichete...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchCtrl.text.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchCtrl.clear();
                          ref.read(labelsProvider.notifier).setSearch('');
                        },
                      )
                    : null,
              ),
              onChanged: (v) => ref.read(labelsProvider.notifier).setSearch(v),
            ),
          ),

          // Status filter chips
          SizedBox(
            height: 48,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding:
                  const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
              children: [
                _FilterChip(
                  label: 'Toate',
                  selected: state.statusFilter == null,
                  onSelected: (_) =>
                      ref.read(labelsProvider.notifier).setStatusFilter(null),
                ),
                ...LabelStatusValue.values.map(
                  (s) => _FilterChip(
                    label: statusLabel(s),
                    selected: state.statusFilter == s,
                    color: statusColor(s),
                    onSelected: (_) =>
                        ref.read(labelsProvider.notifier).setStatusFilter(s),
                  ),
                ),
              ],
            ),
          ),

          Expanded(child: _buildContent(context, state)),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.go('/camera'),
        icon: const Icon(Icons.camera_alt),
        label: const Text('Nouă etichetă'),
      ),
    );
  }

  Widget _buildContent(BuildContext context, LabelsListState state) {
    if (state.isLoading && state.items.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.errorMessage != null && state.items.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 48, color: Colors.black38),
            const SizedBox(height: 12),
            Text(state.errorMessage!,
                style: const TextStyle(color: Colors.black54)),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () => ref.read(labelsProvider.notifier).refresh(),
              child: const Text('Reîncercați'),
            ),
          ],
        ),
      );
    }

    if (state.items.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.label_off_outlined,
                size: 64, color: Colors.black26),
            const SizedBox(height: 16),
            const Text(
              'Nicio etichetă găsită.',
              style: TextStyle(color: Colors.black54, fontSize: 16),
            ),
            const SizedBox(height: 8),
            const Text(
              'Apăsați butonul + pentru a adăuga una.',
              style: TextStyle(color: Colors.black38, fontSize: 13),
            ),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: () => ref.read(labelsProvider.notifier).refresh(),
      child: ListView.builder(
        padding: const EdgeInsets.all(12),
        itemCount: state.items.length,
        itemBuilder: (_, i) => _LabelCard(
          item: state.items[i],
          onTap: () => context.go('/labels/${state.items[i].id}'),
          onDelete: () => _confirmDelete(context, state.items[i].id),
        ),
      ),
    );
  }

  Future<void> _confirmDelete(BuildContext context, String id) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Ștergeți eticheta?'),
        content: const Text('Această acțiune nu poate fi anulată.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Anulare'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            style: TextButton.styleFrom(foregroundColor: AppColors.error),
            child: const Text('Șterge'),
          ),
        ],
      ),
    );
    if (confirmed == true) {
      ref.read(labelsProvider.notifier).delete(id);
    }
  }
}

// ---------------------------------------------------------------------------
// Widgets
// ---------------------------------------------------------------------------

class _FilterChip extends StatelessWidget {
  const _FilterChip({
    required this.label,
    required this.selected,
    required this.onSelected,
    this.color,
  });

  final String label;
  final bool selected;
  final void Function(bool) onSelected;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: FilterChip(
        label: Text(label, style: const TextStyle(fontSize: 12)),
        selected: selected,
        onSelected: onSelected,
        backgroundColor: color?.withValues(alpha: 0.4),
        selectedColor:
            color?.withValues(alpha: 0.7) ??
            AppColors.primary.withValues(alpha: 0.2),
      ),
    );
  }
}

class _LabelCard extends StatelessWidget {
  const _LabelCard({
    required this.item,
    required this.onTap,
    required this.onDelete,
  });

  final LabelListItem item;
  final VoidCallback onTap;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final df = DateFormat('dd.MM.yyyy HH:mm', 'ro_RO');

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              // Thumbnail
              ClipRRect(
                borderRadius: BorderRadius.circular(8),
                child: SizedBox(
                  width: 60,
                  height: 60,
                  child: item.thumbnailUrl != null
                      ? Image.network(
                          item.thumbnailUrl!,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) => _placeholder(),
                        )
                      : _placeholder(),
                ),
              ),
              const SizedBox(width: 12),

              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      item.productName ??
                          'Etichetă ${item.id.substring(0, 8)}',
                      style: const TextStyle(
                          fontWeight: FontWeight.w600, fontSize: 15),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        _StatusChip(status: item.status),
                        const Spacer(),
                        Text(
                          df.format(item.createdAt.toLocal()),
                          style: const TextStyle(
                              fontSize: 11, color: Colors.black45),
                        ),
                      ],
                    ),
                  ],
                ),
              ),

              IconButton(
                icon: const Icon(Icons.delete_outline,
                    color: Colors.black38, size: 20),
                onPressed: onDelete,
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _placeholder() => Container(
        color: Colors.grey[200],
        child: const Icon(Icons.image_outlined,
            color: Colors.black26, size: 30),
      );
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status});

  final LabelStatusValue status;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: statusColor(status),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        statusLabel(status),
        style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w500),
      ),
    );
  }
}
