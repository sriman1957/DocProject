import 'package:flutter/material.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../../auth/models/user.dart';
import '../../groups/models/group.dart';
import '../../subgroups/models/subgroup.dart';
import '../controllers/subgroup_submissions_controller.dart';
import '../models/vault_document.dart';
import '../services/documents_service.dart';
import '../services/file_download_service.dart';
import '../widgets/subgroup_document_card.dart';

class SubgroupSubmissionsScreen extends StatefulWidget {
  const SubgroupSubmissionsScreen({
    required this.user,
    required this.parentGroup,
    required this.subgroup,
    this.controller,
    this.autoLoad = true,
    super.key,
  });

  final User user;
  final Group parentGroup;
  final Subgroup subgroup;
  final SubgroupSubmissionsController? controller;
  final bool autoLoad;

  @override
  State<SubgroupSubmissionsScreen> createState() =>
      _SubgroupSubmissionsScreenState();
}

class _SubgroupSubmissionsScreenState extends State<SubgroupSubmissionsScreen> {
  late final SubgroupSubmissionsController _controller;

  @override
  void initState() {
    super.initState();
    _controller =
        widget.controller ??
        SubgroupSubmissionsController(
          documentsService: DocumentsService(
            apiClient: ApiClient(),
            tokenStorage: TokenStorage(),
          ),
          groupId: widget.subgroup.groupId,
          subgroupId: widget.subgroup.id,
          fileDownloadService: const PlatformFileDownloadService(),
        );

    if (widget.autoLoad && widget.user.isFaculty) {
      _controller.loadSubmissions();
    }
  }

  Future<void> _handleDownload(VaultDocument doc) async {
    try {
      final savedPath = await _controller.downloadDocument(doc);
      if (!mounted) return;

      if (savedPath != null) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Saved "${doc.originalFilename}" successfully.'),
            backgroundColor: Colors.green.shade800,
          ),
        );
      }
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(e.message),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            'Failed to download "${doc.originalFilename}". Please try again.',
          ),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    // Strict role check: only faculty and college admins can access
    if (!widget.user.isFaculty) {
      return Scaffold(
        appBar: AppBar(title: const Text('Submissions Review')),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.lock_rounded, size: 48, color: colors.error),
                const SizedBox(height: 16),
                Text(
                  'Faculty Access Only',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Subgroup submissions are strictly restricted to authorized faculty members.',
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(widget.subgroup.name),
        actions: [
          IconButton(
            key: const Key('refresh_submissions_button'),
            icon: const Icon(Icons.refresh_rounded),
            tooltip: 'Refresh submissions',
            onPressed: () => _controller.refresh(),
          ),
        ],
      ),
      body: SafeArea(
        child: Column(
          children: [
            // Context header banner
            Container(
              width: double.infinity,
              margin: const EdgeInsets.fromLTRB(16, 8, 16, 12),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: colors.primaryContainer.withValues(alpha: 0.35),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(
                  color: colors.primary.withValues(alpha: 0.2),
                ),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    Icons.folder_shared_rounded,
                    color: colors.primary,
                    size: 20,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Parent Group: ${widget.parentGroup.name}',
                          style: theme.textTheme.labelMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: colors.primary,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'Reviewing submissions for subgroup #${widget.subgroup.id}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: colors.onPrimaryContainer,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),

            // Main content area
            Expanded(
              child: ListenableBuilder(
                listenable: _controller,
                builder: (context, _) {
                  return switch (_controller.state) {
                    SubmissionsInitial() => const SizedBox.shrink(),
                    SubmissionsLoading() => const Center(
                      child: CircularProgressIndicator(
                        key: Key('submissions_loading_indicator'),
                      ),
                    ),
                    SubmissionsLoaded(:final documents) => _buildDocumentList(
                      context,
                      documents,
                    ),
                    SubmissionsEmpty() => _buildEmptyState(context),
                    SubmissionsError(:final message, :final canRetry) =>
                      _buildErrorState(context, message, canRetry),
                  };
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDocumentList(
    BuildContext context,
    List<VaultDocument> documents,
  ) {
    return RefreshIndicator(
      onRefresh: () => _controller.refresh(),
      child: ListView.separated(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        itemCount: documents.length,
        separatorBuilder: (context, _) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          final doc = documents[index];
          final isDownloading = _controller.isDownloading(doc.id);

          return SubgroupDocumentCard(
            document: doc,
            isDownloading: isDownloading,
            onDownload: () => _handleDownload(doc),
          );
        },
      ),
    );
  }

  Widget _buildEmptyState(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Center(
      key: const Key('submissions_empty_state'),
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: colors.surfaceContainerHighest,
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.folder_open_rounded,
                size: 40,
                color: colors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            Text(
              'No Submissions Yet',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              'No documents have been submitted to this subgroup yet.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: colors.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            OutlinedButton.icon(
              onPressed: () => _controller.refresh(),
              icon: const Icon(Icons.refresh_rounded, size: 18),
              label: const Text('Refresh'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildErrorState(BuildContext context, String message, bool canRetry) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Center(
      key: const Key('submissions_error_state'),
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: colors.errorContainer,
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.error_outline_rounded,
                size: 36,
                color: colors.onErrorContainer,
              ),
            ),
            const SizedBox(height: 18),
            Text(
              'Unable to load submissions',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              message,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: colors.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            if (canRetry) ...[
              const SizedBox(height: 20),
              FilledButton.icon(
                onPressed: () => _controller.refresh(),
                icon: const Icon(Icons.refresh_rounded, size: 18),
                label: const Text('Try Again'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
