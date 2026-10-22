import 'package:flutter/material.dart';

import '../../../core/widgets/app_placeholder_screen.dart';
import '../../auth/auth_controller.dart';
import '../../auth/models/user.dart';
import '../../documents/controllers/document_upload_controller.dart';
import '../../documents/documents_controller.dart';
import '../../documents/screens/personal_vault_screen.dart';
import '../../documents/services/documents_service.dart';
import '../../groups/groups_controller.dart';
import '../../groups/screens/groups_listing_screen.dart';
import '../../groups/services/groups_service.dart';
import '../../subgroups/services/subgroups_service.dart';
import '../../subgroups/subgroups_controller.dart';
import 'faculty_dashboard_screen.dart';
import 'student_dashboard_screen.dart';

class AppShell extends StatefulWidget {
  const AppShell({
    required this.user,
    required this.authController,
    this.groupsController,
    this.subgroupsController,
    this.documentsController,
    this.uploadController,
    super.key,
  });

  final User user;
  final AuthController authController;
  final GroupsController? groupsController;
  final SubgroupsController? subgroupsController;
  final DocumentsController? documentsController;
  final DocumentUploadController? uploadController;

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  int _currentIndex = 0;
  late final GroupsController _groupsController;
  late final SubgroupsController _subgroupsController;
  late final DocumentsController _documentsController;
  late final DocumentUploadController _uploadController;

  @override
  void initState() {
    super.initState();
    _groupsController =
        widget.groupsController ??
        GroupsController(
          groupsService: GroupsService(
            apiClient: widget.authController.authService.apiClient,
            tokenStorage: widget.authController.tokenStorage,
          ),
        );
    _subgroupsController =
        widget.subgroupsController ??
        SubgroupsController(
          subgroupsService: SubgroupsService(
            apiClient: widget.authController.authService.apiClient,
            tokenStorage: widget.authController.tokenStorage,
          ),
        );
    _documentsController =
        widget.documentsController ??
        DocumentsController(
          documentsService: DocumentsService(
            apiClient: widget.authController.authService.apiClient,
            tokenStorage: widget.authController.tokenStorage,
          ),
        );
    _uploadController =
        widget.uploadController ??
        DocumentUploadController(
          documentsService: _documentsController.documentsService,
        );
  }

  void _onSelectTab(int index) {
    setState(() {
      _currentIndex = index;
    });

    final isStudent = widget.user.userRole.isStudent;
    if (isStudent && index == 1) {
      _documentsController.loadDocuments();
    }

    final groupsTabIndex = isStudent ? 2 : 1;
    if (index == groupsTabIndex) {
      _groupsController.loadGroups();
    }
  }

  Future<void> _confirmLogout(BuildContext context) async {
    final shouldLogout = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Sign out'),
        content: const Text('Are you sure you want to sign out of DocProject?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: const Text('Sign out'),
          ),
        ],
      ),
    );

    if (shouldLogout == true) {
      await widget.authController.logout();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isStudent = widget.user.userRole.isStudent;

    final pages = isStudent
        ? [
            StudentDashboardScreen(
              user: widget.user,
              onNavigateTab: _onSelectTab,
              onLogout: () => _confirmLogout(context),
            ),
            PersonalVaultScreen(
              user: widget.user,
              controller: _documentsController,
              uploadController: _uploadController,
              autoLoad: false,
            ),
            GroupsListingScreen(
              user: widget.user,
              groupsController: _groupsController,
              subgroupsController: _subgroupsController,
              uploadController: _uploadController,
              autoLoad: false,
            ),
            const AppPlaceholderScreen(
              title: 'Notifications',
              icon: Icons.notifications_outlined,
              description: 'Stay updated with active submission windows and certificate collection alerts.',
            ),
          ]
        : [
            FacultyDashboardScreen(
              user: widget.user,
              onNavigateTab: _onSelectTab,
              onLogout: () => _confirmLogout(context),
            ),
            GroupsListingScreen(
              user: widget.user,
              groupsController: _groupsController,
              subgroupsController: _subgroupsController,
              autoLoad: false,
            ),
            _FacultySubgroupsHubTab(onNavigateToGroups: () => _onSelectTab(1)),
            const AppPlaceholderScreen(
              title: 'Notifications',
              icon: Icons.notifications_outlined,
              description: 'Monitor student submissions and department document collection updates.',
            ),
          ];

    final destinations = isStudent
        ? const [
            NavigationDestination(
              icon: Icon(Icons.dashboard_outlined),
              selectedIcon: Icon(Icons.dashboard),
              label: 'Dashboard',
            ),
            NavigationDestination(
              icon: Icon(Icons.folder_shared_outlined),
              selectedIcon: Icon(Icons.folder_shared),
              label: 'Vault',
            ),
            NavigationDestination(
              icon: Icon(Icons.groups_outlined),
              selectedIcon: Icon(Icons.groups),
              label: 'Groups',
            ),
            NavigationDestination(
              icon: Icon(Icons.notifications_outlined),
              selectedIcon: Icon(Icons.notifications),
              label: 'Alerts',
            ),
          ]
        : const [
            NavigationDestination(
              icon: Icon(Icons.dashboard_outlined),
              selectedIcon: Icon(Icons.dashboard),
              label: 'Dashboard',
            ),
            NavigationDestination(
              icon: Icon(Icons.groups_outlined),
              selectedIcon: Icon(Icons.groups),
              label: 'Groups',
            ),
            NavigationDestination(
              icon: Icon(Icons.folder_copy_outlined),
              selectedIcon: Icon(Icons.folder_copy),
              label: 'Subgroups',
            ),
            NavigationDestination(
              icon: Icon(Icons.notifications_outlined),
              selectedIcon: Icon(Icons.notifications),
              label: 'Alerts',
            ),
          ];

    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            Icon(Icons.description_rounded, color: colors.primary),
            const SizedBox(width: 8),
            const Text(
              'DocProject',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
          ],
        ),
        actions: [
          Container(
            margin: const EdgeInsets.only(right: 8),
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: colors.primaryContainer,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Text(
              isStudent ? 'Student' : 'Faculty',
              style: theme.textTheme.labelSmall?.copyWith(
                color: colors.onPrimaryContainer,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.logout_rounded),
            tooltip: 'Sign out',
            onPressed: () => _confirmLogout(context),
          ),
        ],
      ),
      body: SafeArea(
        child: IndexedStack(index: _currentIndex, children: pages),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: _onSelectTab,
        destinations: destinations,
      ),
    );
  }
}

class _FacultySubgroupsHubTab extends StatelessWidget {
  const _FacultySubgroupsHubTab({required this.onNavigateToGroups});

  final VoidCallback onNavigateToGroups;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 480),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 72,
                height: 72,
                decoration: BoxDecoration(
                  color: colors.secondaryContainer,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.folder_copy_rounded,
                  size: 36,
                  color: colors.onSecondaryContainer,
                ),
              ),
              const SizedBox(height: 20),
              Text(
                'Collection Subgroups',
                style: theme.textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 10),
              Text(
                'Subgroups are organized under specific batch groups. Select any group from Groups Management to browse its collection activities and submission windows.',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: colors.onSurfaceVariant,
                  height: 1.4,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              FilledButton.icon(
                onPressed: onNavigateToGroups,
                icon: const Icon(Icons.groups_rounded),
                label: const Text('Browse Assigned Groups'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
