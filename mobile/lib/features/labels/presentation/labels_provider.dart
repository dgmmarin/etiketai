import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/labels_repository.dart';
import '../domain/label.dart';
import '../domain/label_detail.dart';
import '../domain/label_status.dart';

// ---------------------------------------------------------------------------
// Labels list
// ---------------------------------------------------------------------------

class LabelsListState {
  const LabelsListState({
    this.items = const [],
    this.isLoading = false,
    this.errorMessage,
    this.searchQuery = '',
    this.statusFilter,
  });

  final List<LabelListItem> items;
  final bool isLoading;
  final String? errorMessage;
  final String searchQuery;
  final LabelStatusValue? statusFilter;

  LabelsListState copyWith({
    List<LabelListItem>? items,
    bool? isLoading,
    String? errorMessage,
    String? searchQuery,
    LabelStatusValue? statusFilter,
    bool clearError = false,
    bool clearFilter = false,
  }) {
    return LabelsListState(
      items: items ?? this.items,
      isLoading: isLoading ?? this.isLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      searchQuery: searchQuery ?? this.searchQuery,
      statusFilter: clearFilter ? null : (statusFilter ?? this.statusFilter),
    );
  }
}

class LabelsNotifier extends StateNotifier<LabelsListState> {
  LabelsNotifier(this._repo) : super(const LabelsListState()) {
    load();
  }

  final LabelsRepository _repo;

  Future<void> load() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final items = await _repo.list(
        search: state.searchQuery.isEmpty ? null : state.searchQuery,
        status: state.statusFilter,
      );
      state = state.copyWith(items: items, isLoading: false);
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        errorMessage: 'Nu s-au putut încărca etichetele.',
      );
    }
  }

  Future<void> refresh() => load();

  void setSearch(String query) {
    state = state.copyWith(searchQuery: query);
    load();
  }

  void setStatusFilter(LabelStatusValue? status) {
    state = state.copyWith(statusFilter: status, clearFilter: status == null);
    load();
  }

  Future<void> delete(String id) async {
    await _repo.delete(id);
    state = state.copyWith(
      items: state.items.where((l) => l.id != id).toList(),
    );
  }
}

final labelsProvider =
    StateNotifierProvider<LabelsNotifier, LabelsListState>((ref) {
  return LabelsNotifier(ref.watch(labelsRepositoryProvider));
});

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

class UploadState {
  const UploadState({
    this.isUploading = false,
    this.label,
    this.errorMessage,
  });

  final bool isUploading;
  final LabelDetail? label;
  final String? errorMessage;
}

class UploadNotifier extends StateNotifier<UploadState> {
  UploadNotifier(this._repo) : super(const UploadState());

  final LabelsRepository _repo;

  Future<LabelDetail?> upload(File file) async {
    state = const UploadState(isUploading: true);
    try {
      final label = await _repo.upload(file);
      state = UploadState(label: label);
      return label;
    } catch (e) {
      state = const UploadState(
        errorMessage: 'Eroare la încărcarea imaginii.',
      );
      return null;
    }
  }

  void reset() => state = const UploadState();
}

final uploadProvider =
    StateNotifierProvider<UploadNotifier, UploadState>((ref) {
  return UploadNotifier(ref.watch(labelsRepositoryProvider));
});

// ---------------------------------------------------------------------------
// Label detail + status polling
// ---------------------------------------------------------------------------

final labelDetailProvider =
    FutureProvider.family<LabelDetail, String>((ref, id) async {
  final repo = ref.watch(labelsRepositoryProvider);
  return repo.get(id);
});

final labelStatusProvider =
    FutureProvider.family<LabelStatus, String>((ref, id) async {
  final repo = ref.watch(labelsRepositoryProvider);
  return repo.getStatus(id);
});
