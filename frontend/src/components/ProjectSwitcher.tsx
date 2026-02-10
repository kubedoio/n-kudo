import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Check, ChevronDown, FolderOpen, Plus, ArrowLeftRight } from 'lucide-react';
import { useProjects, useProject } from '@/api/hooks';
import { switchProject } from '@/api/auth';
import { toast } from '@/stores/toastStore';
import { cn } from '@/utils/cn';

export function ProjectSwitcher() {
  const navigate = useNavigate();
  const [isOpen, setIsOpen] = useState(false);
  const [isSwitching, setIsSwitching] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  
  // Get all projects
  const { data: projects, isLoading: isLoadingProjects } = useProjects();
  
  // Get current project
  const { data: currentProject, isLoading: isLoadingCurrent } = useProject('current');
  
  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);
  
  const handleSwitchProject = async (projectId: string) => {
    if (projectId === currentProject?.id) {
      setIsOpen(false);
      return;
    }
    
    setIsSwitching(true);
    try {
      await switchProject(projectId);
      toast.success('Switched project successfully');
      setIsOpen(false);
      // Reload to refresh all data
      window.location.reload();
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Failed to switch project');
    } finally {
      setIsSwitching(false);
    }
  };
  
  const isLoading = isLoadingProjects || isLoadingCurrent;
  
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 bg-white/10 rounded-lg">
        <div className="w-5 h-5 rounded bg-white/20 animate-pulse" />
        <div className="w-24 h-4 rounded bg-white/20 animate-pulse" />
      </div>
    );
  }
  
  // If user has only one project, show simple display
  if (projects && projects.length === 1) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 bg-white/10 rounded-lg">
        <FolderOpen className="w-5 h-5 text-white" />
        <span className="text-sm font-medium text-white truncate max-w-[150px]">
          {currentProject?.name || projects[0]?.name}
        </span>
      </div>
    );
  }
  
  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        disabled={isSwitching}
        className={cn(
          "flex items-center gap-2 px-3 py-2 rounded-lg transition-colors",
          "bg-white/10 hover:bg-white/20 text-white",
          isOpen && "bg-white/20",
          isSwitching && "opacity-70 cursor-not-allowed"
        )}
      >
        <FolderOpen className="w-5 h-5" />
        <span className="text-sm font-medium truncate max-w-[150px] hidden md:block">
          {currentProject?.name || 'Select Project'}
        </span>
        {isSwitching ? (
          <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
        ) : (
          <ChevronDown className={cn("w-4 h-4 transition-transform", isOpen && "rotate-180")} />
        )}
      </button>
      
      {isOpen && (
        <div className="absolute top-full left-0 mt-2 w-72 bg-white rounded-lg shadow-xl border border-gray-200 z-50 overflow-hidden">
          {/* Header */}
          <div className="px-4 py-3 border-b border-gray-100 bg-gray-50">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
              Your Projects
            </p>
          </div>
          
          {/* Project List */}
          <div className="max-h-64 overflow-y-auto">
            {projects?.map((project) => (
              <button
                key={project.id}
                onClick={() => handleSwitchProject(project.id)}
                className={cn(
                  "w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-gray-50 transition-colors",
                  currentProject?.id === project.id && "bg-blue-50/50"
                )}
              >
                <div className={cn(
                  "w-8 h-8 rounded-lg flex items-center justify-center shrink-0",
                  currentProject?.id === project.id 
                    ? "bg-blue-100 text-blue-600" 
                    : "bg-gray-100 text-gray-500"
                )}>
                  <FolderOpen className="w-4 h-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className={cn(
                    "text-sm font-medium truncate",
                    currentProject?.id === project.id ? "text-blue-900" : "text-gray-900"
                  )}>
                    {project.name}
                  </p>
                  <p className="text-xs text-gray-500 truncate">
                    {project.slug}
                  </p>
                </div>
                {currentProject?.id === project.id && (
                  <Check className="w-4 h-4 text-blue-600 shrink-0" />
                )}
              </button>
            ))}
          </div>
          
          {/* Footer */}
          <div className="border-t border-gray-100 p-2">
            <button
              onClick={() => {
                setIsOpen(false);
                navigate('/projects');
              }}
              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
            >
              <ArrowLeftRight className="w-4 h-4" />
              Manage Projects
            </button>
            <button
              onClick={() => {
                setIsOpen(false);
                navigate('/projects');
              }}
              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-blue-600 hover:bg-blue-50 rounded-md transition-colors mt-1"
            >
              <Plus className="w-4 h-4" />
              Create New Project
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
