#!/usr/bin/env python3
"""
Script to remove duplicate type declarations from types.go
Keeps the newer, more comprehensive versions and removes older ones.
"""

import re

def remove_duplicate_declarations():
    with open('types.go', 'r') as f:
        lines = f.readlines()
    
    print(f"Processing {len(lines)} lines...")
    
    # Define the ranges of duplicate declarations to remove (keep the newer ones)
    # Based on the build errors, we need to remove:
    # 1. ServiceModelAPI struct (keep the interface)
    # 2. ServiceModelRegistry struct (keep the consolidated one)
    
    # Find the line numbers of duplicates
    duplicates_to_remove = []
    
    i = 0
    while i < len(lines):
        line = lines[i].strip()
        
        # Remove older ServiceModelAPI struct (around line 422)
        if line.startswith('type ServiceModelAPI struct'):
            print(f"Found ServiceModelAPI struct at line {i+1}")
            # Find the closing brace
            start_line = i
            brace_count = 0
            found_opening = False
            
            while i < len(lines):
                if '{' in lines[i]:
                    found_opening = True
                    brace_count += lines[i].count('{')
                    brace_count -= lines[i].count('}')
                elif '}' in lines[i] and found_opening:
                    brace_count -= lines[i].count('}')
                    brace_count += lines[i].count('{')
                
                i += 1
                if found_opening and brace_count == 0:
                    end_line = i
                    print(f"  Marking lines {start_line+1}-{end_line} for removal")
                    duplicates_to_remove.extend(range(start_line, end_line))
                    break
            continue
        
        # Remove older ServiceModelRegistry struct (around line 432)  
        elif line.startswith('type ServiceModelRegistry struct') and i < 650:  # Keep the later one
            print(f"Found older ServiceModelRegistry struct at line {i+1}")
            start_line = i
            brace_count = 0
            found_opening = False
            
            while i < len(lines):
                if '{' in lines[i]:
                    found_opening = True
                    brace_count += lines[i].count('{')
                    brace_count -= lines[i].count('}')
                elif '}' in lines[i] and found_opening:
                    brace_count -= lines[i].count('}')
                    brace_count += lines[i].count('{')
                
                i += 1
                if found_opening and brace_count == 0:
                    end_line = i
                    print(f"  Marking lines {start_line+1}-{end_line} for removal")
                    duplicates_to_remove.extend(range(start_line, end_line))
                    break
            continue
        
        i += 1
    
    # Remove the duplicate lines
    print(f"Removing {len(duplicates_to_remove)} duplicate lines...")
    filtered_lines = [line for i, line in enumerate(lines) if i not in duplicates_to_remove]
    
    # Write the cleaned file
    with open('types.go', 'w') as f:
        f.writelines(filtered_lines)
    
    print(f"Cleaned file now has {len(filtered_lines)} lines")
    return len(lines) - len(filtered_lines)

if __name__ == '__main__':
    removed_count = remove_duplicate_declarations()
    print(f"Successfully removed {removed_count} duplicate lines")