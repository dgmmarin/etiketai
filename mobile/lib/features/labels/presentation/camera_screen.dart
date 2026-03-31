import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_image_compress/flutter_image_compress.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';

import '../../../core/theme/app_theme.dart';
import 'labels_provider.dart';

class CameraScreen extends ConsumerStatefulWidget {
  const CameraScreen({super.key});

  @override
  ConsumerState<CameraScreen> createState() => _CameraScreenState();
}

class _CameraScreenState extends ConsumerState<CameraScreen> {
  final _picker = ImagePicker();
  bool _isProcessing = false;

  Future<void> _pickImage(ImageSource source) async {
    final xFile = await _picker.pickImage(
      source: source,
      imageQuality: 90,
      maxWidth: 2000,
      maxHeight: 2000,
    );
    if (xFile == null) return;

    setState(() => _isProcessing = true);
    try {
      final compressed = await _compressImage(File(xFile.path));
      if (!mounted) return;
      await _uploadAndNavigate(compressed);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Eroare la procesarea imaginii.')),
        );
      }
    } finally {
      if (mounted) setState(() => _isProcessing = false);
    }
  }

  Future<File> _compressImage(File source) async {
    final dir = await getTemporaryDirectory();
    final targetPath =
        '${dir.path}/compressed_${DateTime.now().millisecondsSinceEpoch}.jpg';
    final result = await FlutterImageCompress.compressAndGetFile(
      source.path,
      targetPath,
      quality: 85,
      minWidth: 800,
      minHeight: 600,
    );
    if (result == null) return source;
    // Ensure under 2 MB
    final file = File(result.path);
    if (await file.length() > 2 * 1024 * 1024) {
      final smaller = await FlutterImageCompress.compressAndGetFile(
        source.path,
        targetPath.replaceAll('.jpg', '_2.jpg'),
        quality: 60,
        minWidth: 800,
        minHeight: 600,
      );
      if (smaller != null) return File(smaller.path);
    }
    return file;
  }

  Future<void> _uploadAndNavigate(File file) async {
    final label = await ref.read(uploadProvider.notifier).upload(file);
    if (!mounted) return;
    if (label != null) {
      context.go('/labels/${label.id}');
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Eroare la încărcarea etichetei. Încercați din nou.'),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        title: const Text('Fotografiați eticheta'),
      ),
      body: Stack(
        children: [
          // Guide overlay
          Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(height: 40),
                Stack(
                  alignment: Alignment.center,
                  children: [
                    // Rectangle guide
                    Container(
                      width: 280,
                      height: 180,
                      decoration: BoxDecoration(
                        border: Border.all(
                          color: AppColors.primary,
                          width: 2,
                        ),
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    // Corner markers
                    ..._buildCorners(280, 180),
                    const Icon(
                      Icons.crop_free,
                      color: Colors.white38,
                      size: 40,
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                const Text(
                  'Poziționați eticheta în cadrul dreptunghiului',
                  style: TextStyle(color: Colors.white70, fontSize: 14),
                  textAlign: TextAlign.center,
                ),
              ],
            ),
          ),

          // Loading overlay
          if (_isProcessing)
            Container(
              color: Colors.black54,
              child: const Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(color: Colors.white),
                    SizedBox(height: 16),
                    Text(
                      'Se procesează imaginea...',
                      style: TextStyle(color: Colors.white),
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
      bottomNavigationBar: Container(
        color: Colors.black,
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceEvenly,
          children: [
            _ActionButton(
              icon: Icons.photo_library_outlined,
              label: 'Galerie',
              onTap: _isProcessing
                  ? null
                  : () => _pickImage(ImageSource.gallery),
            ),
            _CameraButton(
              onTap: _isProcessing
                  ? null
                  : () => _pickImage(ImageSource.camera),
            ),
            _ActionButton(
              icon: Icons.info_outline,
              label: 'Sfaturi',
              onTap: () => _showTips(context),
            ),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildCorners(double w, double h) {
    const size = 20.0;
    const thickness = 3.0;
    final color = AppColors.primary;
    return [
      // Top-left
      Positioned(
        left: (MediaQuery.sizeOf(context).width - w) / 2 - w / 2,
        top: -(h / 2) + 0,
        child: Container(
          width: size,
          height: thickness,
          color: color,
        ),
      ),
    ];
  }

  void _showTips(BuildContext ctx) {
    showModalBottomSheet(
      context: ctx,
      builder: (_) => const _TipsSheet(),
    );
  }
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.icon,
    required this.label,
    this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: onTap == null ? Colors.white38 : Colors.white, size: 28),
          const SizedBox(height: 4),
          Text(
            label,
            style: TextStyle(
              color: onTap == null ? Colors.white38 : Colors.white70,
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}

class _CameraButton extends StatelessWidget {
  const _CameraButton({this.onTap});

  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 72,
        height: 72,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          border: Border.all(color: Colors.white, width: 3),
          color: onTap == null ? Colors.white38 : Colors.white,
        ),
        child: Icon(
          Icons.camera_alt,
          size: 36,
          color: onTap == null ? Colors.white54 : Colors.black,
        ),
      ),
    );
  }
}

class _TipsSheet extends StatelessWidget {
  const _TipsSheet();

  @override
  Widget build(BuildContext context) {
    final tips = [
      'Asigurați-vă că eticheta este bine luminată.',
      'Evitați umbrele sau reflexiile pe etichetă.',
      'Mențineți telefonul paralel cu eticheta.',
      'Includeți toată eticheta în cadru.',
      'Rezoluția minimă recomandată: 800×600 px.',
    ];

    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Sfaturi pentru o imagine bună',
            style: Theme.of(context)
                .textTheme
                .titleLarge
                ?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          ...tips.map(
            (t) => Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Icon(Icons.check_circle_outline,
                      size: 18, color: AppColors.secondary),
                  const SizedBox(width: 8),
                  Expanded(child: Text(t)),
                ],
              ),
            ),
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}
